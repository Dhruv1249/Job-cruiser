import { ATSAdapter } from './adapter-interface';
import { AshbyAdapter } from './ashby-adapter';
import { GenericAdapter } from './generic-adapter';
import { GreenhouseAdapter } from './greenhouse-adapter';
import { LeverAdapter } from './lever-adapter';

/**
 * Registry managing candidate ATS platform adapters and detecting the active adapter.
 */
export class AdapterRegistry {
  private readonly adapters: ATSAdapter[];
  private readonly fallbackAdapter: ATSAdapter;

  public constructor(customAdapters?: ATSAdapter[]) {
    this.fallbackAdapter = new GenericAdapter();
    this.adapters = customAdapters ?? [
      new GreenhouseAdapter(),
      new LeverAdapter(),
      new AshbyAdapter(),
    ];
  }

  /**
   * Evaluates registered adapters in order, returning the first adapter whose
   * detection criteria matches the current webpage, or the generic fallback adapter.
   */
  public detectActiveAdapter(): ATSAdapter {
    for (const adapter of this.adapters) {
      if (adapter.detect()) {
        return adapter;
      }
    }

    return this.fallbackAdapter;
  }
}
