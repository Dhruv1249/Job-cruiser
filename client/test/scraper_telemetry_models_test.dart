import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_app/models/scraper_telemetry_models.dart';

void main() {
  group('ScraperTelemetryData Parsing', () {
    test('parses full telemetry JSON payload correctly', () {
      final mockPayload = {
        'total_jobs': 12500,
        'jobs_last_24h': 420,
        'jobs_last_7d': 2850,
        'unique_companies': 1400,
        'kpis': {
          'total_jobs': 12500,
          'jobs_last_24h': 420,
          'jobs_last_7d': 2850,
          'unique_companies': 1400,
          'evaluated_jobs_count': 9200,
          'evaluation_coverage_pct': 73.6,
          'overall_avg_match_score': 76.4,
          'remote_jobs_count': 5100,
          'remote_jobs_pct': 40.8,
          'top_volume_source': 'linkedin',
          'top_quality_source': 'greenhouse',
        },
        'sources_volume': [
          {
            'source': 'linkedin',
            'total_jobs': 6200,
            'jobs_last_24h': 210,
            'jobs_last_7d': 1400,
            'remote_jobs': 2100,
            'onsite_jobs': 4100,
            'share_pct': 49.6,
          }
        ],
        'sources_quality': [
          {
            'source': 'greenhouse',
            'evaluated_count': 1200,
            'avg_score': 84.5,
            'elite_matches': 540,
            'good_matches': 480,
            'low_matches': 180,
            'high_match_yield_pct': 45.0,
          }
        ],
        'ingestion_timeline': [
          {
            'date': '2026-09-01',
            'jobs_count': 310,
          }
        ],
        'score_distribution': {
          'tier_90_100': 1200,
          'tier_80_89': 2400,
          'tier_60_79': 3800,
          'tier_below_60': 1800,
          'unevaluated_count': 3300,
          'avg_score': 76.4,
        },
        'run_health': {
          'total_runs_recorded': 45,
          'successful_runs': 42,
          'failed_runs': 3,
          'success_rate_pct': 93.3,
          'avg_duration_seconds': 185,
        },
        'top_companies': [
          {
            'company_name': 'Acme Corp',
            'job_count': 42,
          }
        ],
        'runs': [
          {
            'run_id': 'test-run-123',
            'started_at': '2026-09-04T12:00:00Z',
            'finished_at': '2026-09-04T12:03:00Z',
            'status': 'completed',
            'jobs_added': 125,
            'sources_hit': '["linkedin", "greenhouse"]',
            'duration_seconds': 180,
            'error_message': '',
          }
        ],
      };

      final data = ScraperTelemetryData.fromJson(mockPayload);

      expect(data.kpis.totalJobs, equals(12500));
      expect(data.kpis.topVolumeSource, equals('linkedin'));
      expect(data.kpis.topQualitySource, equals('greenhouse'));
      expect(data.sourcesVolume.length, equals(1));
      expect(data.sourcesVolume.first.source, equals('linkedin'));
      expect(data.sourcesQuality.length, equals(1));
      expect(data.sourcesQuality.first.highMatchYieldPct, equals(45.0));
      expect(data.ingestionTimeline.length, equals(1));
      expect(data.scoreDistribution.tier90To100, equals(1200));
      expect(data.runHealth.successRatePct, equals(93.3));
      expect(data.topCompanies.length, equals(1));
      expect(data.runs.length, equals(1));
      expect(data.runs.first.sourcesList, contains('greenhouse'));
    });

    test('handles empty or malformed payload gracefully', () {
      final data = ScraperTelemetryData.fromJson({});

      expect(data.kpis.totalJobs, equals(0));
      expect(data.sourcesVolume, isEmpty);
      expect(data.sourcesQuality, isEmpty);
      expect(data.ingestionTimeline, isEmpty);
      expect(data.runs, isEmpty);
    });
  });
}
