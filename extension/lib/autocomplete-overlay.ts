import { OverlaySuggestion } from './types';

export type { OverlaySuggestion };

/**
 * Renders an isolated closed shadow root overlay displaying autofill suggestions below focused inputs.
 * Encapsulates styling within the shadow DOM to prevent ATS website interference and inspection.
 */
export class AutocompleteOverlay {
  private onSuggestionSelected: (
    suggestion: OverlaySuggestion,
    targetElement: HTMLElement
  ) => void;
  private hostElement: HTMLElement | null = null;
  private shadowRoot: ShadowRoot | null = null;
  private overlayContainer: HTMLElement | null = null;
  private suggestionsListContainer: HTMLElement | null = null;
  private currentTargetElement: HTMLElement | null = null;
  private currentSuggestions: readonly OverlaySuggestion[] = [];
  private highlightedIndex: number = -1;
  private focusOutTimeoutIdentifier: number | null = null;
  private boundKeyboardHandler: ((event: KeyboardEvent) => void) | null = null;
  private boundPointerDownHandler: ((event: MouseEvent) => void) | null = null;
  private boundFocusOutHandler: ((event: FocusEvent) => void) | null = null;

  /**
   * Initializes a new instance of AutocompleteOverlay.
   *
   * @param onSuggestionSelected - Callback invoked when a user chooses an autofill suggestion.
   */
  public constructor(
    onSuggestionSelected: (
      suggestion: OverlaySuggestion,
      targetElement: HTMLElement
    ) => void
  ) {
    this.onSuggestionSelected = onSuggestionSelected;
  }

  /**
   * Updates the callback invoked when a suggestion is selected.
   *
   * @param onSuggestionSelected - The replacement suggestion selection callback.
   */
  public setSelectionHandler(
    onSuggestionSelected: (
      suggestion: OverlaySuggestion,
      targetElement: HTMLElement
    ) => void
  ): void {
    this.onSuggestionSelected = onSuggestionSelected;
  }

  /**
   * Displays the autocomplete suggestions card positioned directly below the target form field.
   *
   * @param targetElement - The active HTML form field receiving user focus.
   * @param suggestions - The collection of candidate profile suggestions to display.
   */
  public show(
    targetElement: HTMLElement,
    suggestions: OverlaySuggestion[]
  ): void {
    if (this.focusOutTimeoutIdentifier !== null) {
      window.clearTimeout(this.focusOutTimeoutIdentifier);
      this.focusOutTimeoutIdentifier = null;
    }

    if (suggestions.length === 0) {
      this.hide();
      return;
    }

    if (
      !this.shadowRoot ||
      !this.overlayContainer ||
      !this.hostElement ||
      !this.hostElement.isConnected
    ) {
      this.buildShadowContainer();
    }

    this.currentTargetElement = targetElement;
    this.currentSuggestions = suggestions;
    this.highlightedIndex = 0;

    this.renderSuggestions(suggestions);
    this.positionBelowElement(targetElement);

    if (this.overlayContainer) {
      this.overlayContainer.style.display = 'block';
    }

    this.attachEventListeners(targetElement);
  }

  /**
   * Hides the autocomplete suggestion overlay and removes active interaction event listeners.
   */
  public hide(): void {
    if (this.focusOutTimeoutIdentifier !== null) {
      window.clearTimeout(this.focusOutTimeoutIdentifier);
      this.focusOutTimeoutIdentifier = null;
    }

    this.removeEventListeners();

    if (this.overlayContainer) {
      this.overlayContainer.style.display = 'none';
    }

    this.currentTargetElement = null;
    this.currentSuggestions = [];
    this.highlightedIndex = -1;
  }

  /**
   * Destroys the overlay component, cleans up event handlers, and removes the shadow host from the DOM.
   */
  public destroy(): void {
    this.hide();

    if (this.hostElement && this.hostElement.parentElement) {
      this.hostElement.parentElement.removeChild(this.hostElement);
    }

    this.hostElement = null;
    this.shadowRoot = null;
    this.overlayContainer = null;
    this.suggestionsListContainer = null;
  }

  /**
   * Checks whether the autocomplete suggestions card is currently visible on the screen.
   *
   * @returns True if the overlay card is active and rendered, false otherwise.
   */
  public isVisible(): boolean {
    return Boolean(
      this.overlayContainer && this.overlayContainer.style.display !== 'none'
    );
  }

  private buildShadowContainer(): void {
    const existingHost = document.getElementById('jc-quickfill-host');
    if (existingHost && existingHost.parentElement) {
      existingHost.parentElement.removeChild(existingHost);
    }

    const host = document.createElement('div');
    host.id = 'jc-quickfill-host';
    host.style.position = 'fixed';
    host.style.top = '0';
    host.style.left = '0';
    host.style.width = '0';
    host.style.height = '0';
    host.style.overflow = 'visible';
    host.style.pointerEvents = 'none';
    host.style.zIndex = '2147483647';
    host.style.display = 'block';

    const mountTarget = document.documentElement || document.body;
    mountTarget.appendChild(host);
    this.hostElement = host;

    const shadow = host.attachShadow({ mode: 'closed' });
    this.shadowRoot = shadow;

    this.injectOverlayStyles();

    const overlayCard = document.createElement('div');
    overlayCard.className = 'overlay-card';
    overlayCard.setAttribute('role', 'listbox');
    overlayCard.setAttribute('aria-label', 'Job Cruiser Quick Fill suggestions');
    overlayCard.style.display = 'none';

    const header = document.createElement('div');
    header.className = 'overlay-header';

    const logoBadge = document.createElement('span');
    logoBadge.className = 'logo-badge';
    const logoSvg = parseSvgMarkup(
      '<svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor" stroke="none"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>'
    );
    if (logoSvg) {
      logoBadge.appendChild(logoSvg);
    }
    logoBadge.appendChild(document.createTextNode(' Quick Fill'));
    header.appendChild(logoBadge);

    const suggestionsList = document.createElement('div');
    suggestionsList.className = 'suggestions-list';
    suggestionsList.setAttribute('role', 'presentation');

    overlayCard.appendChild(header);
    overlayCard.appendChild(suggestionsList);
    shadow.appendChild(overlayCard);

    this.overlayContainer = overlayCard;
    this.suggestionsListContainer = suggestionsList;
  }

  private renderSuggestions(suggestions: OverlaySuggestion[]): void {
    if (!this.suggestionsListContainer) {
      return;
    }

    this.suggestionsListContainer.replaceChildren();

    suggestions.forEach((suggestion, index) => {
      const rowElement = document.createElement('div');
      rowElement.className = 'suggestion-row';
      rowElement.setAttribute('role', 'option');
      rowElement.setAttribute('data-index', String(index));
      rowElement.setAttribute(
        'aria-selected',
        index === this.highlightedIndex ? 'true' : 'false'
      );

      if (index === this.highlightedIndex) {
        rowElement.classList.add('highlighted');
      }

      const iconElement = document.createElement('span');
      iconElement.className = 'suggestion-icon';
      const iconSvg = parseSvgMarkup(getIconSvgMarkup(suggestion.icon));
      if (iconSvg) {
        iconElement.appendChild(iconSvg);
      }

      const contentContainer = document.createElement('div');
      contentContainer.className = 'suggestion-content';

      const primaryTextElement = document.createElement('span');
      primaryTextElement.className = 'suggestion-primary';
      primaryTextElement.textContent = suggestion.primaryText;
      contentContainer.appendChild(primaryTextElement);

      if (suggestion.secondaryText) {
        const secondaryTextElement = document.createElement('span');
        secondaryTextElement.className = 'suggestion-secondary';
        secondaryTextElement.textContent = suggestion.secondaryText;
        contentContainer.appendChild(secondaryTextElement);
      }

      rowElement.appendChild(iconElement);
      rowElement.appendChild(contentContainer);

      rowElement.addEventListener('mouseenter', () => {
        this.highlightedIndex = index;
        this.updateHighlightedRowVisuals();
      });

      rowElement.addEventListener('mousedown', (event) => {
        event.preventDefault();
      });

      rowElement.addEventListener('click', () => {
        if (this.currentTargetElement) {
          const target = this.currentTargetElement;
          this.onSuggestionSelected(suggestion, target);
        }
        this.hide();
      });

      this.suggestionsListContainer?.appendChild(rowElement);
    });
  }

  private updateHighlightedRowVisuals(): void {
    if (!this.suggestionsListContainer) {
      return;
    }

    const rows = this.suggestionsListContainer.querySelectorAll('.suggestion-row');
    rows.forEach((row, index) => {
      const isHighlighted = index === this.highlightedIndex;
      row.classList.toggle('highlighted', isHighlighted);
      row.setAttribute('aria-selected', isHighlighted ? 'true' : 'false');
    });
  }

  private positionBelowElement(element: HTMLElement): void {
    if (!this.overlayContainer) {
      return;
    }

    const boundingRectangle = element.getBoundingClientRect();
    const horizontalMargin = 8;
    const overlayWidth = 320;
    const estimatedHeight = 220;

    let leftPosition = boundingRectangle.left;
    if (leftPosition + overlayWidth > window.innerWidth - horizontalMargin) {
      leftPosition = Math.max(
        horizontalMargin,
        window.innerWidth - overlayWidth - horizontalMargin
      );
    }

    let topPosition = boundingRectangle.bottom + 4;
    if (
      topPosition + estimatedHeight > window.innerHeight &&
      boundingRectangle.top > estimatedHeight
    ) {
      topPosition = boundingRectangle.top - estimatedHeight - 4;
    }

    this.overlayContainer.style.position = 'fixed';
    this.overlayContainer.style.top = `${Math.max(4, topPosition)}px`;
    this.overlayContainer.style.left = `${Math.max(4, leftPosition)}px`;
  }

  private handleKeyboardNavigation(event: KeyboardEvent): void {
    if (event.defaultPrevented || !this.isVisible() || this.currentSuggestions.length === 0) {
      return;
    }

    if (event.key === 'Escape') {
      event.preventDefault();
      event.stopImmediatePropagation();
      this.hide();
      return;
    }

    if (event.key === 'ArrowDown') {
      event.preventDefault();
      event.stopImmediatePropagation();
      this.highlightedIndex =
        (this.highlightedIndex + 1) % this.currentSuggestions.length;
      this.updateHighlightedRowVisuals();
      return;
    }

    if (event.key === 'ArrowUp') {
      event.preventDefault();
      event.stopImmediatePropagation();
      this.highlightedIndex =
        (this.highlightedIndex - 1 + this.currentSuggestions.length) %
        this.currentSuggestions.length;
      this.updateHighlightedRowVisuals();
      return;
    }

    if (event.key === 'Enter') {
      if (
        this.highlightedIndex >= 0 &&
        this.highlightedIndex < this.currentSuggestions.length &&
        this.currentTargetElement
      ) {
        event.preventDefault();
        event.stopImmediatePropagation();
        const selectedSuggestion = this.currentSuggestions[this.highlightedIndex];
        if (selectedSuggestion) {
          const target = this.currentTargetElement;
          this.onSuggestionSelected(selectedSuggestion, target);
        }
        this.hide();
      }
    }
  }

  private attachEventListeners(targetElement: HTMLElement): void {
    this.removeEventListeners();

    this.boundKeyboardHandler = (event: KeyboardEvent) => {
      this.handleKeyboardNavigation(event);
    };

    this.boundFocusOutHandler = () => {
      this.focusOutTimeoutIdentifier = window.setTimeout(() => {
        this.hide();
      }, 150);
    };

    this.boundPointerDownHandler = (event: MouseEvent) => {
      if (this.hostElement && event.composedPath) {
        const path = event.composedPath();
        if (path.includes(this.hostElement) || path.includes(targetElement)) {
          return;
        }
      }
      this.hide();
    };

    targetElement.addEventListener('keydown', this.boundKeyboardHandler);
    targetElement.addEventListener('focusout', this.boundFocusOutHandler);
    window.addEventListener('keydown', this.boundKeyboardHandler);
    document.addEventListener('pointerdown', this.boundPointerDownHandler);
  }

  private removeEventListeners(): void {
    if (this.focusOutTimeoutIdentifier !== null) {
      window.clearTimeout(this.focusOutTimeoutIdentifier);
      this.focusOutTimeoutIdentifier = null;
    }

    if (this.boundKeyboardHandler) {
      if (this.currentTargetElement) {
        this.currentTargetElement.removeEventListener(
          'keydown',
          this.boundKeyboardHandler
        );
      }
      window.removeEventListener('keydown', this.boundKeyboardHandler);
      this.boundKeyboardHandler = null;
    }

    if (this.boundFocusOutHandler && this.currentTargetElement) {
      this.currentTargetElement.removeEventListener(
        'focusout',
        this.boundFocusOutHandler
      );
      this.boundFocusOutHandler = null;
    }

    if (this.boundPointerDownHandler) {
      document.removeEventListener('pointerdown', this.boundPointerDownHandler);
      this.boundPointerDownHandler = null;
    }
  }

  private injectOverlayStyles(): void {
    if (!this.shadowRoot) {
      return;
    }

    const styleElement = document.createElement('style');
    styleElement.textContent = `
      :host {
        all: initial;
        position: absolute;
        top: 0;
        left: 0;
        z-index: 2147483647;
      }
      .overlay-card {
        position: fixed;
        width: 320px;
        max-width: calc(100vw - 32px);
        background-color: #FFFFFF;
        border: 1px solid #E2E8F0;
        border-radius: 10px;
        box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.12), 0 8px 10px -6px rgba(0, 0, 0, 0.08);
        font-family: 'Inter', system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        font-size: 13px;
        color: #191C1E;
        overflow: hidden;
        z-index: 2147483647;
        box-sizing: border-box;
        pointer-events: auto;
      }
      .overlay-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 8px 12px;
        background-color: #F7F9FB;
        border-bottom: 1px solid #E2E8F0;
      }
      .logo-badge {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        font-size: 11px;
        font-weight: 800;
        color: #191C1E;
        letter-spacing: 0.05em;
        text-transform: uppercase;
      }
      .suggestions-list {
        max-height: 240px;
        overflow-y: auto;
        padding: 4px 0;
      }
      .suggestion-row {
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 9px 14px;
        cursor: pointer;
        transition: background-color 0.12s ease;
        user-select: none;
      }
      .suggestion-row:hover,
      .suggestion-row.highlighted {
        background-color: #F2F4F6;
      }
      .suggestion-icon {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        color: #000000;
        flex-shrink: 0;
        width: 16px;
        height: 16px;
      }
      .suggestion-content {
        display: flex;
        flex-direction: column;
        min-width: 0;
        flex: 1;
      }
      .suggestion-primary {
        font-weight: 600;
        color: #191C1E;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
      .suggestion-secondary {
        font-size: 11px;
        color: #45464D;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        margin-top: 1px;
      }
    `;

    this.shadowRoot.appendChild(styleElement);
  }
}

/**
 * Parses an SVG markup string into a standalone SVG DOM element without using innerHTML.
 */
function parseSvgMarkup(svgMarkup: string): Element | null {
  try {
    const parser = new DOMParser();
    const documentNode = parser.parseFromString(svgMarkup, 'image/svg+xml');
    return documentNode.documentElement;
  } catch {
    return null;
  }
}

/**
 * Returns inline SVG markup for semantic icon names, eliminating all emojis.
 */
function getIconSvgMarkup(iconName: string): string {
  switch (iconName) {
    case 'user':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>`;
    case 'mail':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="20" height="16" x="2" y="4" rx="2"/><path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"/></svg>`;
    case 'phone':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"/></svg>`;
    case 'map-pin':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z"/><circle cx="12" cy="10" r="3"/></svg>`;
    case 'globe':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20"/><path d="M2 12h20"/></svg>`;
    case 'briefcase':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 20V4a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/><rect width="20" height="14" x="2" y="6" rx="2"/></svg>`;
    case 'building':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="16" height="20" x="4" y="2" rx="2"/><path d="M9 22v-4h6v4"/><path d="M8 6h.01"/><path d="M16 6h.01"/><path d="M8 10h.01"/><path d="M16 10h.01"/><path d="M8 14h.01"/><path d="M16 14h.01"/></svg>`;
    case 'graduation-cap':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21.42 10.922a1 1 0 0 0-.019-1.838L12.83 5.18a2 2 0 0 0-1.66 0L2.6 9.08a1 1 0 0 0 0 1.832l8.57 3.908a2 2 0 0 0 1.66 0z"/><path d="M22 10v6"/><path d="M6 12.5V16a6 3 0 0 0 12 0v-3.5"/></svg>`;
    case 'calendar':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="4" rx="2"/><path d="M16 2v4"/><path d="M8 2v4"/><path d="M3 10h18"/></svg>`;
    case 'check':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"/></svg>`;
    case 'clipboard':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="8" height="4" x="8" y="2" rx="1"/><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/></svg>`;
    case 'clock':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>`;
    case 'truck':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 18V6a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2v11a1 1 0 0 0 1 1h2"/><path d="M15 18H9"/><path d="M19 18h2a1 1 0 0 0 1-1v-3.65a1 1 0 0 0-.22-.62l-3.48-4.35A1 1 0 0 0 17.52 8H14v10Z"/><circle cx="17" cy="18.5" r="2.5"/><circle cx="7" cy="18.5" r="2.5"/></svg>`;
    case 'dollar-sign':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>`;
    case 'file-text':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/><path d="M10 9H8"/><path d="M16 13H8"/><path d="M16 17H8"/></svg>`;
    case 'link':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>`;
    case 'tag':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.586 2.586A2 2 0 0 0 11.172 2H4a2 2 0 0 0-2 2v7.172a2 2 0 0 0 .586 1.414l8.704 8.704a2.426 2.426 0 0 0 3.42 0l6.58-6.58a2.426 2.426 0 0 0 0-3.42z"/><circle cx="7.5" cy="7.5" r=".5" fill="currentColor"/></svg>`;
    case 'rotate-cw':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/></svg>`;
    case 'list':
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="8" x2="21" y1="6" y2="6"/><line x1="8" x2="21" y1="12" y2="12"/><line x1="8" x2="21" y1="18" y2="18"/><line x1="3" x2="3.01" y1="6" y2="6"/><line x1="3" x2="3.01" y1="12" y2="12"/><line x1="3" x2="3.01" y1="18" y2="18"/></svg>`;
    default:
      return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" x2="12" y1="8" y2="12"/><line x1="12" x2="12.01" y1="16" y2="16"/></svg>`;
  }
}
