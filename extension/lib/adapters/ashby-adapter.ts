import { FieldClassification, FieldType } from '../types';
import { ATSAdapter, JobPageMetadata } from './adapter-interface';

/**
 * Adapter providing platform detection, field classification enhancement,
 * and page metadata extraction for Ashby applicant tracking system portals.
 */
export class AshbyAdapter implements ATSAdapter {
  public readonly platformName: string = 'Ashby';

  /**
   * Detects whether the current webpage is an Ashby application page or job portal.
   */
  public detect(): boolean {
    if (typeof window === 'undefined' || typeof document === 'undefined') {
      return false;
    }

    const currentHostname = window.location.hostname.toLowerCase();
    if (
      currentHostname.includes('ashbyhq.com') ||
      currentHostname.includes('jobs.ashby.io')
    ) {
      return true;
    }

    const ashbyContainer = document.querySelector(
      '[data-testid*="ashby"], [class*="_ashby_"], #ashby_app',
    );
    return ashbyContainer !== null;
  }

  /**
   * Enhances field classification using Ashby data-testid container and element attributes.
   */
  public enhanceClassification(
    element: HTMLElement,
    baseClassification: FieldClassification,
  ): FieldClassification {
    const directTestId = element.getAttribute('data-testid');
    const containerTestId = element
      .closest('[data-testid]')
      ?.getAttribute('data-testid');
    const testIdentifier = (directTestId ?? containerTestId ?? '').toLowerCase();

    if (testIdentifier.length > 0) {
      const specificFieldType = this.matchAshbyTestIdentifier(testIdentifier);
      if (specificFieldType !== null) {
        return {
          fieldType: specificFieldType,
          confidence: 1.0,
          visibleLabel: baseClassification.visibleLabel,
          inputKind: baseClassification.inputKind,
        };
      }

      if (baseClassification.fieldType !== 'unknown') {
        return {
          fieldType: baseClassification.fieldType,
          confidence: 1.0,
          visibleLabel: baseClassification.visibleLabel,
          inputKind: baseClassification.inputKind,
        };
      }
    }

    return baseClassification;
  }

  /**
   * Extracts job posting metadata including title and company from the Ashby document.
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

  private matchAshbyTestIdentifier(testIdentifier: string): FieldType | null {
    if (
      testIdentifier.includes('first-name') ||
      testIdentifier.includes('firstname')
    ) {
      return 'first_name';
    }

    if (
      testIdentifier.includes('last-name') ||
      testIdentifier.includes('lastname')
    ) {
      return 'last_name';
    }

    if (testIdentifier.includes('name')) {
      return 'full_name';
    }

    if (testIdentifier.includes('email')) {
      return 'email';
    }

    if (testIdentifier.includes('phone')) {
      return 'phone';
    }

    if (testIdentifier.includes('resume') || testIdentifier.includes('cv')) {
      return 'resume_upload';
    }

    if (testIdentifier.includes('linkedin')) {
      return 'linkedin';
    }

    if (testIdentifier.includes('github')) {
      return 'github';
    }

    if (
      testIdentifier.includes('portfolio') ||
      testIdentifier.includes('website')
    ) {
      return 'portfolio';
    }

    if (
      testIdentifier.includes('cover-letter') ||
      testIdentifier.includes('coverletter')
    ) {
      return 'bio_summary';
    }

    return null;
  }

  private extractJobTitle(): string | null {
    const headingElement = document.querySelector(
      'h1[data-testid="job-title"], h1.job-title, h1',
    );
    if (headingElement && headingElement.textContent) {
      const cleanedTitle = headingElement.textContent.trim();
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
    const companyHeading = document.querySelector(
      '[data-testid="company-name"], .company-name',
    );
    if (companyHeading && companyHeading.textContent) {
      const cleanedCompany = companyHeading.textContent.trim();
      if (cleanedCompany.length > 0) {
        return cleanedCompany;
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
      'form, [data-testid*="application-form"], [data-testid*="application"]',
    );
    if (applicationForm !== null) {
      return true;
    }

    return window.location.pathname.includes('/application');
  }
}
