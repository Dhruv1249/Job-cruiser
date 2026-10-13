import { FieldClassification } from '../types';
import { ATSAdapter, JobPageMetadata } from './adapter-interface';

/**
 * Universal fallback ATS adapter used when no specialized applicant tracking system is recognized.
 */
export class GenericAdapter implements ATSAdapter {
  public readonly platformName: string = 'Generic';

  /**
   * Universal adapter detection always succeeds as the fallback.
   */
  public detect(): boolean {
    return true;
  }

  /**
   * Returns base classification unmodified for generic web pages.
   */
  public enhanceClassification(
    _element: HTMLElement,
    baseClassification: FieldClassification,
  ): FieldClassification {
    return baseClassification;
  }

  /**
   * Extracts generic page metadata such as heading and document title.
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

  private extractJobTitle(): string | null {
    const primaryHeading = document.querySelector('h1');
    if (primaryHeading && primaryHeading.textContent) {
      const cleanedHeading = primaryHeading.textContent.trim();
      if (cleanedHeading.length > 0) {
        return cleanedHeading;
      }
    }

    const openGraphTitle = document.querySelector('meta[property="og:title"]');
    if (openGraphTitle) {
      const content = openGraphTitle.getAttribute('content');
      if (content && content.trim().length > 0) {
        return content.trim();
      }
    }

    const documentTitle = document.title.trim();
    if (documentTitle.length > 0) {
      return documentTitle;
    }

    return null;
  }

  private extractCompanyName(): string | null {
    const openGraphSiteName = document.querySelector(
      'meta[property="og:site_name"]',
    );
    if (openGraphSiteName) {
      const content = openGraphSiteName.getAttribute('content');
      if (content && content.trim().length > 0) {
        return content.trim();
      }
    }

    const metaAuthor = document.querySelector('meta[name="author"]');
    if (metaAuthor) {
      const content = metaAuthor.getAttribute('content');
      if (content && content.trim().length > 0) {
        return content.trim();
      }
    }

    return null;
  }

  private detectApplicationPage(): boolean {
    const applicationInputs = document.querySelectorAll(
      'form input[type="text"], form input[type="email"], form textarea',
    );
    return applicationInputs.length > 0;
  }
}
