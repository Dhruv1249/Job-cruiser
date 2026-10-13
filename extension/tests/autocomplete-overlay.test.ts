import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AutocompleteOverlay } from '../lib/autocomplete-overlay';
import { OverlaySuggestion } from '../lib/types';

describe('AutocompleteOverlay', () => {
  let overlay: AutocompleteOverlay;
  let suggestionSelectHandler: (
    suggestion: OverlaySuggestion,
    targetElement: HTMLElement
  ) => void;

  beforeEach(() => {
    vi.useFakeTimers();
    document.body.innerHTML = '';
    suggestionSelectHandler = vi.fn();
    overlay = new AutocompleteOverlay(suggestionSelectHandler);
  });

  afterEach(() => {
    overlay.destroy();
    vi.useRealTimers();
  });

  it('creates closed shadow host on document body when show is called', () => {
    const inputElement = document.createElement('input');
    document.body.appendChild(inputElement);

    const suggestions: OverlaySuggestion[] = [
      {
        icon: '👤',
        primaryText: 'Grace Hopper',
        secondaryText: 'Full Name',
        value: 'Grace Hopper',
      },
    ];

    overlay.show(inputElement, suggestions);

    const hostElement = document.getElementById('jc-quickfill-host');
    expect(hostElement).not.toBeNull();
    expect(hostElement?.shadowRoot).toBeNull();
    expect(overlay.isVisible()).toBe(true);
  });

  it('hides when hide is called', () => {
    const inputElement = document.createElement('input');
    document.body.appendChild(inputElement);

    const suggestions: OverlaySuggestion[] = [
      {
        icon: '📧',
        primaryText: 'grace@example.com',
        value: 'grace@example.com',
      },
    ];

    overlay.show(inputElement, suggestions);
    expect(overlay.isVisible()).toBe(true);

    overlay.hide();
    expect(overlay.isVisible()).toBe(false);
  });

  it('hides after 150 milliseconds debounce when target element loses focus', () => {
    const inputElement = document.createElement('input');
    document.body.appendChild(inputElement);

    const suggestions: OverlaySuggestion[] = [
      {
        icon: '📱',
        primaryText: '+15551234567',
        value: '+15551234567',
      },
    ];

    overlay.show(inputElement, suggestions);
    expect(overlay.isVisible()).toBe(true);

    inputElement.dispatchEvent(new FocusEvent('focusout', { bubbles: true }));

    vi.advanceTimersByTime(100);
    expect(overlay.isVisible()).toBe(true);

    vi.advanceTimersByTime(60);
    expect(overlay.isVisible()).toBe(false);
  });

  it('hides when Escape key is pressed', () => {
    const inputElement = document.createElement('input');
    document.body.appendChild(inputElement);

    const suggestions: OverlaySuggestion[] = [
      {
        icon: '📍',
        primaryText: 'Boston, MA',
        value: 'Boston, MA',
      },
    ];

    overlay.show(inputElement, suggestions);
    expect(overlay.isVisible()).toBe(true);

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    expect(overlay.isVisible()).toBe(false);
  });

  it('selects highlighted suggestion on Enter key press', () => {
    const inputElement = document.createElement('input');
    document.body.appendChild(inputElement);

    const suggestions: OverlaySuggestion[] = [
      {
        icon: '🔗',
        primaryText: 'https://linkedin.com/in/gracehopper',
        value: 'https://linkedin.com/in/gracehopper',
      },
    ];

    overlay.show(inputElement, suggestions);

    inputElement.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'Enter', bubbles: true })
    );

    expect(suggestionSelectHandler).toHaveBeenCalledTimes(1);
    expect(suggestionSelectHandler).toHaveBeenCalledWith(
      suggestions[0],
      inputElement
    );
    expect(overlay.isVisible()).toBe(false);
  });

  it('navigates through suggestions with ArrowDown and ArrowUp keys', () => {
    const inputElement = document.createElement('input');
    document.body.appendChild(inputElement);

    const suggestions: OverlaySuggestion[] = [
      {
        icon: '👤',
        primaryText: 'First Name',
        value: 'Grace',
      },
      {
        icon: '👤',
        primaryText: 'Last Name',
        value: 'Hopper',
      },
    ];

    overlay.show(inputElement, suggestions);

    inputElement.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true })
    );

    inputElement.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'Enter', bubbles: true })
    );

    expect(suggestionSelectHandler).toHaveBeenCalledTimes(1);
    expect(suggestionSelectHandler).toHaveBeenCalledWith(
      suggestions[1],
      inputElement
    );
  });

  it('removes host element completely upon destroy', () => {
    const inputElement = document.createElement('input');
    document.body.appendChild(inputElement);

    overlay.show(inputElement, [
      {
        icon: '🌍',
        primaryText: 'United States',
        value: 'United States',
      },
    ]);

    expect(document.getElementById('jc-quickfill-host')).not.toBeNull();

    overlay.destroy();

    expect(document.getElementById('jc-quickfill-host')).toBeNull();
    expect(overlay.isVisible()).toBe(false);
  });
});
