import json
import os
from playwright.sync_api import sync_playwright
from typing import List, Optional
from pydantic import BaseModel, Field
from bs4 import BeautifulSoup
import google.generativeai as genai
import time

class CompanyExtraction(BaseModel):
    name: str = Field(description="Name of the company")
    domain: Optional[str] = Field(None, description="Website domain of the company, e.g., stripe.com")
    description: Optional[str] = Field(None, description="Brief description of what the company does")
    industry: Optional[str] = Field(None, description="Industry the company operates in")
    company_size: Optional[str] = Field(None, description="Estimated number of employees, e.g., 10-50, 1000+")
    hq_location: Optional[str] = Field(None, description="Headquarters location of the company")
    poc_name: Optional[str] = None
    poc_email: Optional[str] = None

class JobExtraction(BaseModel):
    title: str = Field(description="The official job title")
    location: Optional[str] = Field(None, description="Where the job is located")
    salary: Optional[str] = Field(None, description="Salary range or exact salary mentioned")
    experience_required: Optional[str] = Field(None, description="Years of experience required, e.g., '3-5 years'")
    job_type: Optional[str] = Field(None, description="e.g., Full-time, Part-time, Contract")
    is_remote: bool = Field(default=False, description="True if the job is remote or work-from-home")
    tags: List[str] = Field(default_factory=list, description="List of technical skills or keywords (e.g., ['Python', 'React', 'AWS'])")

class CombinedExtraction(BaseModel):
    company: CompanyExtraction
    job: JobExtraction

genai.configure(api_key=os.environ.get("GEMINI_API_KEY"))
model = genai.GenerativeModel('gemini-2.5-flash')

def extract_structured_data(raw_html: str, url: str, source: str) -> dict:
    print(f"Scraping from {url}...")
    
    soup = BeautifulSoup(raw_html, "html.parser")
    
    for script in soup(["script", "style", "nav", "footer"]):
        script.extract()
        
    clean_text = soup.get_text(separator="\n", strip=True)
    
    prompt = f"""
    You are an expert data extraction assistant. I am providing you with the raw text scraped from a job posting.
    Source URL: {url}
    Job Board Source: {source}
    
    Extract the company information and job details from the text below. 
    If a field is not mentioned in the text, leave it as null/None. Do not make up information.
    For 'tags', extract the top 5-10 most important technical skills or domain keywords.
    
    Raw Job Description Text:
    ---
    {clean_text[:10000]} 
    ---
    """
    
    response = model.generate_content(
        prompt,
        generation_config=genai.GenerationConfig(
            response_mime_type="application/json",
            response_schema=CombinedExtraction,
            temperature=0.1, 
        ),
    )
    
    extracted_data = json.loads(response.text)
    
    extracted_data['job']['url'] = url
    extracted_data['job']['source'] = source
    extracted_data['job']['raw_desc'] = clean_text 
    
    return extracted_data


# PLAYWRIGHT
def fetch_page_html(url: str, page=None) -> str:
    """
    Fetches the HTML of a page. If a Playwright 'page' object is provided, 
    it reuses it (faster for batching). Otherwise, it spins up a new browser.
    """
    print(f"Fetching HTML for {url}...")
    
    if page:
        page.goto(url, wait_until="domcontentloaded", timeout=30000)
        return page.content()
        
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        new_page = browser.new_page()
        new_page.goto(url, wait_until="domcontentloaded", timeout=30000)
        html = new_page.content()
        browser.close()
        return html


def process_job_url(url: str, source_name: str, page=None) -> dict:

    raw_html = fetch_page_html(url, page=page)
    
    structured_data = extract_structured_data(raw_html, url, source_name)
    
    return structured_data

def process_multiple_urls(job_listings: List[dict]) -> List[dict]:
    
    results = []
    
    with sync_playwright() as p:
        print("Launching browser for batch processing...")
        browser = p.chromium.launch(headless=True)
        page = browser.new_page()
        
        for item in job_listings:
            url = item.get("url")
            source = item.get("source", "Unknown")
            
            try:
                print(f"\n--- Processing: {url} ---")
                data = process_job_url(url, source, page=page)
                results.append(data)
                
                time.sleep(3) 
                
            except Exception as e:
                print(f"Error processing {url}: {e}")
                
        browser.close()
        
    return results

if __name__ == "__main__":

    TARGET_JOBS = [
        {"url": "https://www.remote3.co", "source": "remote3"},
        {"url": "https://www.hirebasis.com", "source": "HireBasis"},
    ]
    
    print(f"Starting batch data retrieval for {len(TARGET_JOBS)} URLs...")
    
    all_results = process_multiple_urls(TARGET_JOBS)
    
    print(f"\nBATCH RETRIEVAL SUCCESSFUL ({len(all_results)}/{len(TARGET_JOBS)} scraped) ---\n")
    print(json.dumps(all_results, indent=2))