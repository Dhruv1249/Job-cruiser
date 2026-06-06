import asyncio
import logging
import re
import json
from urllib.parse import urljoin, urlparse
from bs4 import BeautifulSoup
from playwright.async_api import async_playwright, TimeoutError as PlaywrightTimeoutError

# Configure basic logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

class UniversalJobScraper:
    def __init__(self, base_url, max_pages=50, max_depth=3):
        """
        Initializes the scraper with the target job site.
        Edge Cases Handled: Limits maximum pages and depth to prevent infinite loops.
        """
        self.base_url = base_url
        self.domain = urlparse(base_url).netloc
        self.max_pages = max_pages
        self.max_depth = max_depth
        
        self.visited_urls = set()
        self.job_listings = []
        
        # Heuristics to identify if a URL is likely a job detail page
        self.job_url_patterns = re.compile(r'(/jobs?/|/careers?/|/position/|/opening/|jobId=|-job-|-role-)', re.IGNORECASE)
        
    async def init_browser(self):
        """Sets up the headless browser with realistic parameters to bypass basic anti-bot software."""
        self.playwright = await async_playwright().start()
        # Launch chromium. Edge Case: Use standard viewport and user-agent to look like a real user.
        self.browser = await self.playwright.chromium.launch(headless=True)
        self.context = await self.browser.new_context(
            viewport={'width': 1920, 'height': 1080},
            user_agent='Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
            ignore_https_errors=True
        )

    async def close_browser(self):
        """Cleans up browser resources."""
        await self.context.close()
        await self.browser.close()
        await self.playwright.stop()

    def is_internal_link(self, url):
        """Edge Case: Ensure the crawler doesn't wander off to external sites (like Twitter or Facebook)."""
        return self.domain in urlparse(url).netloc

    async def crawl(self, start_url):
        """
        Breadth-first asynchronous crawler.
        Edge Cases: Network timeouts, bad links, dynamic loading.
        """
        queue = [(start_url, 0)]  # (url, depth)
        
        page = await self.context.new_page()

        while queue and len(self.visited_urls) < self.max_pages:
            current_url, depth = queue.pop(0)

            # Edge Case: Prevent duplicate visits and trailing slash discrepancies
            normalized_url = current_url.rstrip('/')
            if normalized_url in self.visited_urls or depth > self.max_depth:
                continue

            self.visited_urls.add(normalized_url)
            logging.info(f"Crawling (Depth {depth}): {normalized_url}")

            try:
                # Edge Case: "networkidle" often times out on modern sites due to continuous background trackers.
                # Changed to "domcontentloaded" and increased the timeout to 60 seconds to be safer.
                await page.goto(current_url, timeout=60000, wait_until="domcontentloaded")
                
                # Human-like delay to avoid rate limiting and allow JavaScript frameworks to render data
                await asyncio.sleep(4)
                
                html_content = await page.content()
                soup = BeautifulSoup(html_content, 'html.parser')

                # 1. If it looks like a job page, extract data
                if self.job_url_patterns.search(normalized_url):
                    job_data = self.extract_job_details(soup, normalized_url)
                    if job_data['title']:  # Only save if we actually found a title
                        self.job_listings.append(job_data)
                        logging.info(f"Found Job: {job_data['title']} at {job_data['company']}")

                # 2. Extract links for further crawling
                if depth < self.max_depth:
                    links = soup.find_all('a', href=True)
                    for link in links:
                        href = link['href']
                        # Edge Case: Handle relative URLs (e.g., href="/about")
                        absolute_url = urljoin(current_url, href).split('#')[0] # Remove fragment identifiers
                        
                        if self.is_internal_link(absolute_url) and absolute_url not in self.visited_urls:
                            queue.append((absolute_url, depth + 1))

            except PlaywrightTimeoutError:
                logging.warning(f"Timeout while loading {current_url}. Skipping.")
            except Exception as e:
                logging.error(f"Error processing {current_url}: {e}")

        await page.close()

    def extract_job_details(self, soup, url):
        """
        Heuristic-based extractor.
        Edge Case: Sites use different HTML tags. We check JSON-LD first (industry standard), 
        then fall back to common HTML class names.
        """
        job_data = {
            'url': url,
            'title': None,
            'company': None,
            'location': None,
            'description': None
        }

        # Strategy 1: Check for JSON-LD structured data (Google Jobs standard)
        script_tags = soup.find_all('script', type='application/ld+json')
        for tag in script_tags:
            try:
                data = json.loads(tag.string if tag.string else "")
                # Handle both dicts and lists of dicts
                if isinstance(data, list):
                    for item in data:
                        if item.get('@type') == 'JobPosting':
                            data = item
                            break
                
                if isinstance(data, dict) and data.get('@type') == 'JobPosting':
                    job_data['title'] = data.get('title')
                    job_data['company'] = data.get('hiringOrganization', {}).get('name')
                    job_data['description'] = BeautifulSoup(data.get('description', ''), 'html.parser').get_text(strip=True)
                    return job_data  # Best quality data, return immediately
            except json.JSONDecodeError:
                continue

        # Strategy 2: Fallback to HTML DOM heuristics if JSON-LD isn't present
        
        # Find Title (Usually an h1)
        h1 = soup.find('h1')
        if h1:
            job_data['title'] = h1.get_text(strip=True)

        # Find Company (Looking for common class names)
        company_tag = soup.find(class_=re.compile(r'company|employer', re.I))
        if company_tag:
            job_data['company'] = company_tag.get_text(strip=True)

        # Find Location
        location_tag = soup.find(class_=re.compile(r'location|city', re.I))
        if location_tag:
            job_data['location'] = location_tag.get_text(strip=True)

        # Find Description (Usually the largest text block on the page)
        description_container = soup.find(class_=re.compile(r'description|details|content', re.I))
        if description_container:
            job_data['description'] = description_container.get_text(separator='\n', strip=True)
        else:
            # Absolute fallback: Get all paragraph text
            paragraphs = soup.find_all('p')
            job_data['description'] = '\n'.join([p.get_text(strip=True) for p in paragraphs if len(p.get_text(strip=True)) > 20])

        return job_data

    def save_to_json(self, filename="job_results.json"):
        """Saves the extracted listings to a JSON file."""
        if not self.job_listings:
            logging.info("No jobs found to save.")
            return

        try:
            with open(filename, 'w', encoding='utf-8') as output_file:
                json.dump(self.job_listings, output_file, ensure_ascii=False, indent=4)
            logging.info(f"Successfully saved {len(self.job_listings)} jobs to {filename}")
        except Exception as e:
            logging.error(f"Failed to save JSON: {e}")

async def main():
    # Example Target: Replace this with the target job board or company career page
    target_url = "https://hirebasis.com/" # Updated to your target
    
    logging.info(f"Starting scraper for {target_url}")
    
    # Increased max_pages to allow the crawler to get past the initial category pages
    scraper = UniversalJobScraper(base_url=target_url, max_pages=100, max_depth=3)
    
    await scraper.init_browser()
    await scraper.crawl(target_url)
    await scraper.close_browser()
    
    scraper.save_to_json("extracted_jobs.json")

if __name__ == "__main__":
    # Run the async loop
    asyncio.run(main())