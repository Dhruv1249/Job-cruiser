import json
import os
import time
import re
from urllib.parse import urljoin, urlparse
from playwright.sync_api import sync_playwright
from bs4 import BeautifulSoup
from typing import List

# ==========================================
# 1. TRADITIONAL EXTRACTION LOGIC (NO AI)
# ==========================================

def extract_comprehensive_data(raw_html: str, url: str, source: str) -> dict:
    """
    Takes raw HTML and uses BeautifulSoup and Regex to extract all possible 
    structured data (lists, headings, metadata, emails) without AI.
    """
    print(f"Scraping from {url} (Traditional Parsing)...")
    
    soup = BeautifulSoup(raw_html, "html.parser")
    
    # 1. Extract Meta Information
    title_tag = soup.find('title')
    page_title = title_tag.get_text(strip=True) if title_tag else None
    
    meta_desc_tag = soup.find('meta', attrs={'name': 'description'})
    meta_desc = meta_desc_tag['content'] if meta_desc_tag else None
    
    # 2. Extract prominent headings (h1, h2) for section titles
    headings = {
        "h1": [h.get_text(strip=True) for h in soup.find_all('h1')],
        "h2": [h.get_text(strip=True) for h in soup.find_all('h2')]
    }
    
    # 3. Clean the HTML for text extraction
    for element in soup(["script", "style", "nav", "footer", "header", "aside"]):
        element.extract()
        
    clean_text = soup.get_text(separator="\n", strip=True)
    
    # 4. Extract standard lists (usually used for Requirements & Benefits)
    structured_lists = []
    for ul in soup.find_all('ul'):
        items = [li.get_text(strip=True) for li in ul.find_all('li') if li.get_text(strip=True)]
        if items:
            structured_lists.append(items)
            
    # 5. Extract contact emails using Regex
    email_pattern = r'[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+'
    emails_found = list(set(re.findall(email_pattern, clean_text)))
    
    # Combine everything into a comprehensive JSON dictionary
    extracted_data = {
        "page_title": page_title,
        "meta_description": meta_desc,
        "headings": headings,
        "structured_lists": structured_lists,
        "contact_emails": emails_found,
        "full_text": clean_text,
        "_metadata": {
            'url': url,
            'source': source,
            'scraped_at': time.strftime('%Y-%m-%dT%H:%M:%SZ'),
            'raw_text_length': len(clean_text)
        }
    }
    
    return extracted_data

# ==========================================
# 2. WEB CRAWLER (PLAYWRIGHT)
# ==========================================
def fetch_page_html(url: str, page=None) -> str:
    """
    Fetches the HTML of a page. If a Playwright 'page' object is provided, 
    it reuses it (faster for batching). Otherwise, it spins up a new browser.
    """
    print(f"Fetching HTML for {url}...")
    
    if page:
        page.goto(url, wait_until="domcontentloaded", timeout=30000)
        return page.content()
        
    # Standalone mode fallback
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        new_page = browser.new_page()
        new_page.goto(url, wait_until="domcontentloaded", timeout=30000)
        html = new_page.content()
        browser.close()
        return html

# ==========================================
# 3. DEEP CRAWLING PIPELINE
# ==========================================
def extract_links(raw_html: str, base_url: str, allowed_domain: str) -> List[str]:
    """
    Finds all href links on a page, normalizes them, and filters them 
    so we only follow internal links (same domain).
    """
    soup = BeautifulSoup(raw_html, "html.parser")
    valid_links = []
    
    for a_tag in soup.find_all("a", href=True):
        href = a_tag["href"]
        # Resolve relative URLs (e.g., /about -> https://company.com/about)
        full_url = urljoin(base_url, href)
        # Strip URL fragments (e.g., #section1) to avoid scraping the same page twice
        full_url = full_url.split("#")[0] 
        
        # Ensure we don't crawl external sites (like Twitter, LinkedIn)
        link_domain = urlparse(full_url).netloc
        if allowed_domain in link_domain:
            valid_links.append(full_url)
            
    return list(set(valid_links)) # Remove duplicates

def crawl_site(start_url: str, source_name: str, page, max_pages: int = 5) -> List[dict]:
    """
    Crawls a starting URL and its subpages up to a max_pages limit.
    Uses Breadth-First Search (BFS) to visit links level by level.
    """
    allowed_domain = urlparse(start_url).netloc
    visited = set()
    queue = [start_url]
    site_results = []
    
    print(f"\n--- Starting Deep Crawl for Domain: {allowed_domain} ---")
    
    while queue and len(visited) < max_pages:
        current_url = queue.pop(0)
        
        if current_url in visited:
            continue
            
        visited.add(current_url)
        
        try:
            # 1. Fetch HTML
            raw_html = fetch_page_html(current_url, page=page)
            
            # 2. Extract structured data from this page
            page_data = extract_comprehensive_data(raw_html, current_url, source_name)
            site_results.append(page_data)
            
            # 3. Find more links to add to our queue
            new_links = extract_links(raw_html, current_url, allowed_domain)
            for link in new_links:
                if link not in visited and link not in queue:
                    queue.append(link)
                    
            print(f"   -> Discovered {len(new_links)} internal links. Queue size: {len(queue)}")
            
            # Polite delay between page requests
            time.sleep(2)
            
        except Exception as e:
            print(f"❌ Error processing {current_url}: {e}")
            
    print(f"--- Finished crawling {allowed_domain} (Crawled {len(visited)} pages) ---")
    return site_results

def process_multiple_urls(starting_urls: List[dict], max_pages_per_url: int = 5) -> List[dict]:
    """
    Efficiently processes multiple seed URLs, deep crawling each one.
    """
    all_results = []
    
    with sync_playwright() as p:
        print("Launching browser for batch deep-crawling...")
        browser = p.chromium.launch(headless=True)
        page = browser.new_page()
        
        for item in starting_urls:
            url = item.get("url")
            source = item.get("source", "Unknown")
            
            # Initiate deep crawl for this specific target
            crawled_data = crawl_site(url, source, page, max_pages=max_pages_per_url)
            all_results.extend(crawled_data)
                
        browser.close()
        
    return all_results

if __name__ == "__main__":
    # Test with a list of URLs
    TARGET_JOBS = [
        {"url": "https://www.remote3.co", "source": "remote3"},
        {"url": "https://www.hirebasis.com", "source": "HireBasis"},
    ]
    
    # max_pages_per_url=3 means it will scrape the initial job post + up to 2 subpages per URL
    print(f"Starting deep crawling for {len(TARGET_JOBS)} seed URLs...")
    
    # Run the batch retrieval pipeline
    all_results = process_multiple_urls(TARGET_JOBS, max_pages_per_url=3)
    
    print(f"\n--- ✅ BATCH RETRIEVAL SUCCESSFUL ({len(all_results)} total pages scraped) ---\n")
    
    # Saving to file instead of printing due to large output
    output_filename = "crawled_data_dump.json"
    with open(output_filename, "w", encoding="utf-8") as f:
        json.dump(all_results, f, indent=2)
    print(f"Data saved to {output_filename}")