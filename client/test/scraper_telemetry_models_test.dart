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

    test('parses sourceDistribution map of numbers and objects correctly', () {
      const logWithCountMap = ScraperRunLog(
        runId: 'run-map-1',
        startedAt: '2026-09-08T10:00:00Z',
        finishedAt: '2026-09-08T10:05:00Z',
        status: 'completed',
        jobsAdded: 205,
        sourcesRaw: '{"greenhouse": 120, "lever": 85}',
        errorMessage: '',
        durationSeconds: 300,
      );

      final distribution = logWithCountMap.sourceDistribution;
      expect(distribution.length, equals(2));
      expect(distribution.first.source, equals('greenhouse'));
      expect(distribution.first.jobsFound, equals(120));
      expect(distribution.last.source, equals('lever'));
      expect(distribution.last.jobsFound, equals(85));
      expect(logWithCountMap.sourcesList, containsAll(['greenhouse', 'lever']));

      const logWithDetailMap = ScraperRunLog(
        runId: 'run-detail-1',
        startedAt: '2026-09-08T10:00:00Z',
        finishedAt: '2026-09-08T10:05:00Z',
        status: 'completed',
        jobsAdded: 150,
        sourcesRaw: '{"ashby": {"jobs_found": 90, "duration_seconds": 14.5}, "workday": {"jobs_found": 60, "duration_seconds": 22.0, "error": "Rate limited"}}',
        errorMessage: '',
        durationSeconds: 300,
      );

      final detailDist = logWithDetailMap.sourceDistribution;
      expect(detailDist.length, equals(2));
      expect(detailDist.first.source, equals('ashby'));
      expect(detailDist.first.jobsFound, equals(90));
      expect(detailDist.first.durationSeconds, equals(14.5));
      expect(detailDist.last.source, equals('workday'));
      expect(detailDist.last.errorMessage, equals('Rate limited'));
    });

    test('aggregates composite granular query keys by platform', () {
      const logWithCompositeKeys = ScraperRunLog(
        runId: 'run-granular-1',
        startedAt: '2026-09-08T10:00:00Z',
        finishedAt: '2026-09-08T10:05:00Z',
        status: 'completed',
        jobsAdded: 350,
        sourcesRaw: '{"linkedin:sde:india": {"jobs_found": 200, "duration_seconds": 15.0}, "linkedin:devops:india": {"jobs_found": 100, "duration_seconds": 10.0}, "indeed:backend:remote": {"jobs_found": 50, "duration_seconds": 4.5}}',
        errorMessage: '',
        durationSeconds: 300,
      );

      final distribution = logWithCompositeKeys.sourceDistribution;
      expect(distribution.length, equals(2));
      expect(distribution.first.source, equals('linkedin'));
      expect(distribution.first.jobsFound, equals(300));
      expect(distribution.first.queryCount, equals(2));
      expect(distribution.first.durationSeconds, equals(25.0));
      expect(distribution.last.source, equals('indeed'));
      expect(distribution.last.jobsFound, equals(50));
      expect(distribution.last.queryCount, equals(1));
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
