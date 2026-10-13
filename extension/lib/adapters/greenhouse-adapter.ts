import { FieldClassification, FieldType } from '../types';
import { ATSAdapter, JobPageMetadata } from './adapter-interface';

/**
 * Adapter providing platform detection, field classification enhancement,
 * and page metadata extraction for Greenhouse ATS application portals.
 */
export class GreenhouseAdapter implements ATSAdapter {
  public readonly platformName: string = 'Greenhouse';

  /**
   * Detects whether the current webpage corresponds to a Greenhouse application or job board.
   */
  public detect(): boolean {
    if (typeof window === 'undefined' || typeof document === 'undefined') {
      return false;
    }

    const currentHostname = window.location.hostname.toLowerCase();
    if (currentHostname.includes('greenhouse.io')) {
      return true;
    }

    const greenhouseInput = document.querySelector(
      'input[name^="job_application"], [id^="job_application"]',
    );
    if (greenhouseInput !== null) {
      return true;
    }

    const greenhouseForm = document.querySelector(
      'form#application_form, form[action*="greenhouse.io"]',
    );
    return greenhouseForm !== null;
  }

  /**
   * Enhances field classification using Greenhouse DOM attributes and naming conventions.
   */
  public enhanceClassification(
    element: HTMLElement,
    baseClassification: FieldClassification,
  ): FieldClassification {
    const attributeName = (element.getAttribute('name') ?? '').toLowerCase();
    const attributeIdentifier = (element.getAttribute('id') ?? '').toLowerCase();

    const specificFieldType = this.matchGreenhouseFieldType(
      attributeName,
      attributeIdentifier,
    );

    if (specificFieldType !== null) {
      return {
        fieldType: specificFieldType,
        confidence: 1.0,
        visibleLabel: baseClassification.visibleLabel,
        inputKind: baseClassification.inputKind,
      };
    }

    if (
      attributeName.startsWith('job_application[') &&
      baseClassification.fieldType !== 'unknown'
    ) {
      return {
        fieldType: baseClassification.fieldType,
        confidence: 1.0,
        visibleLabel: baseClassification.visibleLabel,
        inputKind: baseClassification.inputKind,
      };
    }

    return baseClassification;
  }

  /**
   * Extracts job posting metadata including title and company from the Greenhouse document.
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

  private matchGreenhouseFieldType(
    attributeName: string,
    attributeIdentifier: string,
  ): FieldType | null {
    if (
      attributeName.includes('first_name') ||
      attributeIdentifier.includes('first_name')
    ) {
      return 'first_name';
    }

    if (
      attributeName.includes('last_name') ||
      attributeIdentifier.includes('last_name')
    ) {
      return 'last_name';
    }

    if (
      attributeName.includes('[email]') ||
      attributeIdentifier.endsWith('_email') ||
      attributeIdentifier === 'email'
    ) {
      return 'email';
    }

    if (
      attributeName.includes('[phone]') ||
      attributeIdentifier.endsWith('_phone') ||
      attributeIdentifier === 'phone'
    ) {
      return 'phone';
    }

    if (
      attributeName.includes('[resume]') ||
      attributeIdentifier.includes('resume')
    ) {
      return 'resume_upload';
    }

    if (
      attributeName.includes('linkedin') ||
      attributeIdentifier.includes('linkedin')
    ) {
      return 'linkedin';
    }

    if (
      attributeName.includes('github') ||
      attributeIdentifier.includes('github')
    ) {
      return 'github';
    }

    if (
      attributeName.includes('portfolio') ||
      attributeIdentifier.includes('portfolio') ||
      attributeName.includes('website')
    ) {
      return 'portfolio';
    }

    return null;
  }

  private extractJobTitle(): string | null {
    const titleElement = document.querySelector(
      'h1.app-title, div.job-title, h1.job-title, h1',
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
    const companyElement = document.querySelector(
      'div.company-name, span.company-name, .company-name',
    );
    if (companyElement && companyElement.textContent) {
      const cleanedCompany = companyElement.textContent.trim();
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
      '#application_form, form#application_form, input[name^="job_application"]',
    );
    return applicationForm !== null;
  }
}
