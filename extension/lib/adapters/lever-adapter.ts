import { FieldClassification, FieldType } from '../types';
import { ATSAdapter, JobPageMetadata } from './adapter-interface';

/**
 * Adapter providing platform detection, field classification enhancement,
 * and page metadata extraction for Lever applicant tracking system portals.
 */
export class LeverAdapter implements ATSAdapter {
  public readonly platformName: string = 'Lever';

  /**
   * Detects whether the current webpage is hosted on or powered by Lever.
   */
  public detect(): boolean {
    if (typeof window === 'undefined' || typeof document === 'undefined') {
      return false;
    }

    const currentHostname = window.location.hostname.toLowerCase();
    if (currentHostname.includes('jobs.lever.co')) {
      return true;
    }

    const leverContainer = document.querySelector(
      '.lever-job-site, .lever-application-form, form[action*="lever.co"]',
    );
    return leverContainer !== null;
  }

  /**
   * Enhances field classification using Lever-specific name attributes.
   */
  public enhanceClassification(
    element: HTMLElement,
    baseClassification: FieldClassification,
  ): FieldClassification {
    const attributeName = (element.getAttribute('name') ?? '').trim();

    const specificFieldType = this.matchLeverFieldType(attributeName);
    if (specificFieldType !== null) {
      return {
        fieldType: specificFieldType,
        confidence: 1.0,
        visibleLabel: baseClassification.visibleLabel,
        inputKind: baseClassification.inputKind,
      };
    }

    return baseClassification;
  }

  /**
   * Extracts job posting metadata including title and company from the Lever document.
   */
  public getJobMetadata(): JobPageMetadata | null {
    if (typeof window === 'undefined' || typeof document === 'undefined') {
      return null;
    }

    const jobTitle = this.extractJobTitle();
    const companyName = this.extractCompanyName();
    const jobUrl = window.location.href;
    const isApplicationPage = this.detectApplicationPage();

    return {
      jobTitle,
      companyName,
      jobUrl,
      isApplicationPage,
    };
  }

  private matchLeverFieldType(attributeName: string): FieldType | null {
    const normalizedName = attributeName.toLowerCase();

    if (attributeName === 'name' || normalizedName === 'name') {
      return 'full_name';
    }

    if (attributeName === 'email' || normalizedName === 'email') {
      return 'email';
    }

    if (attributeName === 'phone' || normalizedName === 'phone') {
      return 'phone';
    }

    if (attributeName === 'org' || normalizedName === 'org') {
      return 'current_company';
    }

    if (
      attributeName.includes('urls[LinkedIn]') ||
      normalizedName.includes('linkedin')
    ) {
      return 'linkedin';
    }

    if (
      attributeName.includes('urls[GitHub]') ||
      normalizedName.includes('github')
    ) {
      return 'github';
    }

    if (
      attributeName.includes('urls[Portfolio]') ||
      normalizedName.includes('portfolio')
    ) {
      return 'portfolio';
    }

    if (
      attributeName.includes('urls[Other]') ||
      normalizedName.includes('website')
    ) {
      return 'website';
    }

    if (
      attributeName === 'comments' ||
      normalizedName === 'comments' ||
      normalizedName === 'additional-information'
    ) {
      return 'bio_summary';
    }

    if (attributeName === 'resume' || normalizedName.includes('resume')) {
      return 'resume_upload';
    }

    return null;
  }

  private extractJobTitle(): string | null {
    const titleElement = document.querySelector(
      '.posting-headline h2, .job-title h2, h2.posting-headline, h2, h1',
    );
    if (titleElement && titleElement.textContent) {
      const cleanedTitle = titleElement.textContent.trim();
      if (cleanedTitle.length > 0) {
        return cleanedTitle;
      }
    }

    const documentTitle = document.title.trim();
    if (documentTitle.length > 0) {
      return documentTitle;
    }

    return null;
  }

  private extractCompanyName(): string | null {
    const companyAttributeElement = document.querySelector(
      'body.lever-job-site [data-company], [data-company]',
    );
    if (companyAttributeElement) {
      const companyAttribute = companyAttributeElement.getAttribute('data-company');
      if (companyAttribute && companyAttribute.trim().length > 0) {
        return companyAttribute.trim();
      }
    }

    const logoElement = document.querySelector(
      '.main-header-logo img, .posting-header img',
    );
    if (logoElement) {
      const altText = logoElement.getAttribute('alt');
      if (altText && altText.trim().length > 0) {
        return altText.trim();
      }
    }

    const openGraphSiteName = document.querySelector(
      'meta[property="og:site_name"]',
    );
    if (openGraphSiteName) {
      const content = openGraphSiteName.getAttribute('content');
      if (content && content.trim().length > 0) {
        return content.trim();
      }
    }

    return null;
  }

  private detectApplicationPage(): boolean {
    const applicationForm = document.querySelector(
      '.application-form, form#application-form, [data-qa="btn-apply"]',
    );
    if (applicationForm !== null) {
      return true;
    }

    return window.location.pathname.includes('/apply');
  }
}
