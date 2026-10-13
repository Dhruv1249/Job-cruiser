import { FieldClassification } from '../types';

/**
 * Metadata extracted from an applicant tracking system job posting or application page.
 */
export interface JobPageMetadata {
  jobTitle: string | null;
  companyName: string | null;
  jobUrl: string;
  isApplicationPage: boolean;
}

/**
 * Platform-specific adapter providing ATS detection, classification refinement,
 * and page metadata extraction.
 */
export interface ATSAdapter {
  readonly platformName: string;
  detect(): boolean;
  enhanceClassification(
    element: HTMLElement,
    baseClassification: FieldClassification,
  ): FieldClassification;
  getJobMetadata(): JobPageMetadata | null;
}
