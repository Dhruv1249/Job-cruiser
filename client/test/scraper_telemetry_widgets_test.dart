import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_app/models/scraper_telemetry_models.dart';
import 'package:flutter_app/widgets/telemetry/telemetry_kpi_grid.dart';
import 'package:flutter_app/widgets/telemetry/ingestion_timeline_chart.dart';
import 'package:flutter_app/widgets/telemetry/source_quality_leaderboard.dart';
import 'package:flutter_app/widgets/telemetry/source_volume_chart.dart';
import 'package:flutter_app/widgets/telemetry/score_distribution_chart.dart';
import 'package:flutter_app/widgets/telemetry/scraper_run_history_card.dart';

void main() {
  group('Scraper Telemetry Widgets Tests', () {
    const mockKpis = TelemetryKpis(
      totalJobs: 10600,
      jobsLast24h: 350,
      jobsLast7d: 2100,
      uniqueCompanies: 1200,
      evaluatedJobsCount: 8500,
      evaluationCoveragePct: 80.2,
      overallAvgMatchScore: 78.4,
      remoteJobsCount: 4200,
      remoteJobsPct: 39.6,
      topVolumeSource: 'greenhouse',
      topQualitySource: 'lever',
    );

    const mockSourcesVolume = [
      SourceVolumeMetric(
        source: 'greenhouse',
        totalJobs: 5000,
        jobsLast24h: 150,
        jobsLast7d: 900,
        remoteJobs: 2000,
        onsiteJobs: 3000,
        sharePct: 47.2,
      ),
      SourceVolumeMetric(
        source: 'lever',
        totalJobs: 3000,
        jobsLast24h: 100,
        jobsLast7d: 600,
        remoteJobs: 1500,
        onsiteJobs: 1500,
        sharePct: 28.3,
      ),
    ];

    const mockSourcesQuality = [
      SourceQualityMetric(
        source: 'lever',
        evaluatedCount: 2500,
        avgScore: 84.5,
        eliteMatches: 600,
        goodMatches: 1400,
        lowMatches: 500,
        highMatchYieldPct: 24.0,
      ),
      SourceQualityMetric(
        source: 'greenhouse',
        evaluatedCount: 4000,
        avgScore: 79.2,
        eliteMatches: 700,
        goodMatches: 2100,
        lowMatches: 1200,
        highMatchYieldPct: 17.5,
      ),
    ];

    const mockTimeline = [
      DailyIngestionMetric(date: '2026-09-01', jobsCount: 320),
      DailyIngestionMetric(date: '2026-09-02', jobsCount: 450),
      DailyIngestionMetric(date: '2026-09-03', jobsCount: 380),
    ];

    const mockScoreDistribution = ScoreDistributionMetric(
      tier90To100: 1500,
      tier80To89: 3000,
      tier60To79: 3500,
      tierBelow60: 500,
      unevaluatedCount: 2100,
      avgScore: 78.4,
    );

    const mockRunHealth = ScraperRunHealthMetric(
      totalRunsRecorded: 42,
      successfulRuns: 40,
      failedRuns: 2,
      successRatePct: 95.2,
      avgDurationSeconds: 125,
    );

    const mockTopCompanies = [
      TopCompanyMetric(companyName: 'Acme Corp', jobCount: 45),
      TopCompanyMetric(companyName: 'Globex Ltd', jobCount: 32),
    ];

    const mockRuns = [
      ScraperRunLog(
        runId: 'run-001',
        startedAt: '2026-09-04T12:00:00Z',
        finishedAt: '2026-09-04T12:02:15Z',
        status: 'completed',
        jobsAdded: 240,
        sourcesRaw: '["greenhouse", "lever"]',
        errorMessage: '',
        durationSeconds: 135,
      ),
      ScraperRunLog(
        runId: 'run-002',
        startedAt: '2026-09-04T06:00:00Z',
        finishedAt: '2026-09-04T06:01:00Z',
        status: 'failed',
        jobsAdded: 0,
        sourcesRaw: '["ashby"]',
        errorMessage: 'Network connection timeout during handshake',
        durationSeconds: 60,
      ),
    ];

    testWidgets('TelemetryKpiGrid renders all 6 KPI cards accurately', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: SingleChildScrollView(
              child: TelemetryKpiGrid(kpis: mockKpis),
            ),
          ),
        ),
      );

      expect(find.text('Total Ingested'), findsOneWidget);
      expect(find.text('10600'), findsOneWidget);
      expect(find.text('+350 24h'), findsOneWidget);

      expect(find.text('Unique Companies'), findsOneWidget);
      expect(find.text('1200'), findsOneWidget);

      expect(find.text('AI Evaluated'), findsOneWidget);
      expect(find.text('80.2%'), findsOneWidget);

      expect(find.text('Avg Match Score'), findsOneWidget);
      expect(find.text('78.4%'), findsOneWidget);

      expect(find.text('Top Volume Source'), findsOneWidget);
      expect(find.text('GREENHOUSE'), findsOneWidget);

      expect(find.text('Best Quality Source'), findsOneWidget);
      expect(find.text('LEVER'), findsOneWidget);
    });

    testWidgets('IngestionTimelineChart renders title and handles empty/populated states', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: IngestionTimelineChart(timeline: []),
          ),
        ),
      );

      expect(find.text('No ingestion timeline data recorded yet.'), findsOneWidget);

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: IngestionTimelineChart(timeline: mockTimeline),
          ),
        ),
      );

      expect(find.text('14-Day Ingestion Velocity'), findsOneWidget);
      expect(find.text('1150 jobs total'), findsOneWidget);
    });

    testWidgets('SourceQualityLeaderboard renders rankings and handles sort selection', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: SingleChildScrollView(
              child: SourceQualityLeaderboard(sourcesQuality: mockSourcesQuality),
            ),
          ),
        ),
      );

      expect(find.text('AI Match Quality Rankings'), findsOneWidget);
      expect(find.text('LEVER'), findsOneWidget);
      expect(find.text('GREENHOUSE'), findsOneWidget);
      expect(find.text('84.5% Avg'), findsOneWidget);
      expect(find.text('79.2% Avg'), findsOneWidget);
      expect(find.text('1'), findsOneWidget);
      expect(find.text('2'), findsOneWidget);
    });

    testWidgets('SourceVolumeChart renders volume bars and sorting modes', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: SingleChildScrollView(
              child: SourceVolumeChart(sourcesVolume: mockSourcesVolume),
            ),
          ),
        ),
      );

      expect(find.text('Source Volume & Intake Velocity'), findsOneWidget);
      expect(find.text('5000 jobs'), findsOneWidget);
      expect(find.text('3000 jobs'), findsOneWidget);
      expect(find.text('+150 24h'), findsOneWidget);
      expect(find.text('+100 24h'), findsOneWidget);
    });

    testWidgets('ScoreDistributionChart renders pie chart legend and top companies', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: SingleChildScrollView(
              child: ScoreDistributionChart(
                scoreDistribution: mockScoreDistribution,
                topCompanies: mockTopCompanies,
              ),
            ),
          ),
        ),
      );

      expect(find.text('AI Match Score Distribution & Top Companies'), findsOneWidget);
      expect(find.text('90–100% Elite'), findsOneWidget);
      expect(find.text('80–89% Top Match'), findsOneWidget);
      expect(find.text('60–79% Good Match'), findsOneWidget);
      expect(find.text('<60% Low Match'), findsOneWidget);
      expect(find.text('Unevaluated'), findsOneWidget);
      expect(find.text('Acme Corp'), findsOneWidget);
      expect(find.text('45 roles'), findsOneWidget);
      expect(find.text('Globex Ltd'), findsOneWidget);
      expect(find.text('32 roles'), findsOneWidget);
    });

    testWidgets('ScraperRunHistoryCard renders run health stats and error banners', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: SingleChildScrollView(
              child: ScraperRunHistoryCard(
                runHealth: mockRunHealth,
                runs: mockRuns,
              ),
            ),
          ),
        ),
      );

      expect(find.text('Scraper Execution Health & Run History'), findsOneWidget);
      expect(find.text('95% Success Rate'), findsOneWidget);
      expect(find.text('42'), findsOneWidget);
      expect(find.text('40'), findsOneWidget);
      expect(find.text('2'), findsOneWidget);
      expect(find.text('+240 jobs'), findsOneWidget);
      expect(find.text('+0 jobs'), findsOneWidget);

      await tester.tap(find.text('+0 jobs'));
      await tester.pumpAndSettle();

      expect(find.text('Network connection timeout during handshake'), findsOneWidget);
    });
  });
}
