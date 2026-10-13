/**
 * Service providing reliable value injection into web form inputs, select controls, and file uploads.
 * Bypasses synthetic event listeners by using prototype property setters and firing native events.
 */
export class InputInjector {
  /**
   * Injects text value into an input or textarea element using prototype property setter bypass.
   * Dispatches input and change events to notify reactive front-end frameworks.
   *
   * @param element - The HTMLInputElement or HTMLTextAreaElement receiving the text.
   * @param value - The text string to inject.
   */
  public injectTextValue(
    element: HTMLInputElement | HTMLTextAreaElement,
    value: string
  ): void {
    element.focus();

    const isTextArea = element.tagName.toLowerCase() === 'textarea';
    const prototype = isTextArea
      ? HTMLTextAreaElement.prototype
      : HTMLInputElement.prototype;

    const propertyDescriptor = Object.getOwnPropertyDescriptor(
      prototype,
      'value'
    );
    const nativePropertySetter = propertyDescriptor?.set;

    if (nativePropertySetter) {
      nativePropertySetter.call(element, value);
    } else {
      element.value = value;
    }

    element.dispatchEvent(
      new Event('input', { bubbles: true, composed: true })
    );
    element.dispatchEvent(
      new Event('change', { bubbles: true, composed: true })
    );

    element.blur();
  }

  /**
   * Injects a selected option into a select dropdown element matching target value or text.
   * Supports exact case-insensitive matches followed by partial substring matches.
   *
   * @param selectElement - The HTMLSelectElement receiving the selection.
   * @param targetValue - The target string to match against option values or text labels.
   * @returns True if a matching option was found and selected, false otherwise.
   */
  public injectSelectValue(
    selectElement: HTMLSelectElement,
    targetValue: string
  ): boolean {
    const normalizedTarget = targetValue.trim().toLowerCase();
    if (!normalizedTarget) {
      return false;
    }

    const availableOptions = Array.from(selectElement.options);

    let matchedOption = availableOptions.find((option) => {
      const optionValue = option.value.trim().toLowerCase();
      const optionText = option.textContent?.trim().toLowerCase() ?? '';
      return optionValue === normalizedTarget || optionText === normalizedTarget;
    });

    if (!matchedOption) {
      matchedOption = availableOptions.find((option) => {
        const optionValue = option.value.toLowerCase();
        const optionText = option.textContent?.toLowerCase() ?? '';
        return (
          optionValue.includes(normalizedTarget) ||
          optionText.includes(normalizedTarget)
        );
      });
    }

    if (!matchedOption) {
      return false;
    }

    selectElement.value = matchedOption.value;
    matchedOption.selected = true;

    selectElement.dispatchEvent(
      new Event('input', { bubbles: true, composed: true })
    );
    selectElement.dispatchEvent(
      new Event('change', { bubbles: true, composed: true })
    );

    return true;
  }

  /**
   * Injects a generated or retrieved document file into an HTML file input element.
   * Constructs a File instance, sets the input files via DataTransfer, and dispatches a change event.
   *
   * @param fileInputElement - The HTMLInputElement with type='file' receiving the document.
   * @param pdfBlob - The Blob payload containing document binary data.
   * @param fileName - The desired name of the uploaded document file.
   */
  public async injectFileUpload(
    fileInputElement: HTMLInputElement,
    pdfBlob: Blob,
    fileName: string
  ): Promise<void> {
    const mimeType = pdfBlob.type || 'application/pdf';
    const documentFile = new File([pdfBlob], fileName, { type: mimeType });

    if (typeof DataTransfer !== 'undefined') {
      const dataTransfer = new DataTransfer();
      dataTransfer.items.add(documentFile);
      fileInputElement.files = dataTransfer.files;
    } else {
      Object.defineProperty(fileInputElement, 'files', {
        value: [documentFile],
        writable: true,
        configurable: true,
      });
    }

    fileInputElement.dispatchEvent(
      new Event('change', { bubbles: true, composed: true })
    );
  }
}
