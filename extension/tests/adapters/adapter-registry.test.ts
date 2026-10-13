import { beforeEach, describe, expect, it } from 'vitest';
import { AdapterRegistry } from '../../lib/adapters/adapter-registry';

describe('AdapterRegistry', () => {
  let registry: AdapterRegistry;

  beforeEach(() => {
    registry = new AdapterRegistry();
    document.body.innerHTML = '';
    const happyDomWindow = window as unknown as {
      happyDOM: { setURL: (destinationUrl: string) => void };
    };
    happyDomWindow.happyDOM.setURL('http://localhost:3000/');
  });

  it('detects GreenhouseAdapter when page matches Greenhouse', () => {
    const happyDomWindow = window as unknown as {
      happyDOM: { setURL: (destinationUrl: string) => void };
    };
    happyDomWindow.happyDOM.setURL('https://boards.greenhouse.io/company/job');

    const adapter = registry.detectActiveAdapter();
    expect(adapter.platformName).toBe('Greenhouse');
  });

  it('detects LeverAdapter when page matches Lever', () => {
    const happyDomWindow = window as unknown as {
      happyDOM: { setURL: (destinationUrl: string) => void };
    };
    happyDomWindow.happyDOM.setURL('https://jobs.lever.co/company/job');

    const adapter = registry.detectActiveAdapter();
    expect(adapter.platformName).toBe('Lever');
  });

  it('detects AshbyAdapter when page matches Ashby', () => {
    const happyDomWindow = window as unknown as {
      happyDOM: { setURL: (destinationUrl: string) => void };
    };
    happyDomWindow.happyDOM.setURL('https://jobs.ashby.io/company/job');

    const adapter = registry.detectActiveAdapter();
    expect(adapter.platformName).toBe('Ashby');
  });

  it('falls back to GenericAdapter when no platform matches', () => {
    const happyDomWindow = window as unknown as {
      happyDOM: { setURL: (destinationUrl: string) => void };
    };
    happyDomWindow.happyDOM.setURL('https://careers.unknowncompany.org/openings');

    const adapter = registry.detectActiveAdapter();
    expect(adapter.platformName).toBe('Generic');
  });
});
