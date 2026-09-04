import 'dart:convert';

/// Key summary metrics across the scraping and AI evaluation lifecycle.
class TelemetryKpis {
  const TelemetryKpis({
    required this.totalJobs,
    required this.jobsLast24h,
    required this.jobsLast7d,
    required this.uniqueCompanies,
    required this.evaluatedJobsCount,
    required this.evaluationCoveragePct,
    required this.overallAvgMatchScore,
    required this.remoteJobsCount,
    required this.remoteJobsPct,
    required this.topVolumeSource,
    required this.topQualitySource,
  });

  final int totalJobs;
  final int jobsLast24h;
  final int jobsLast7d;
  final int uniqueCompanies;
  final int evaluatedJobsCount;
  final double evaluationCoveragePct;
  final double overallAvgMatchScore;
  final int remoteJobsCount;
  final double remoteJobsPct;
  final String topVolumeSource;
  final String topQualitySource;

  factory TelemetryKpis.fromJson(Map<String, dynamic> json) {
    return TelemetryKpis(
      totalJobs: (json['total_jobs'] as num?)?.toInt() ?? 0,
      jobsLast24h: (json['jobs_last_24h'] as num?)?.toInt() ?? 0,
      jobsLast7d: (json['jobs_last_7d'] as num?)?.toInt() ?? 0,
      uniqueCompanies: (json['unique_companies'] as num?)?.toInt() ?? 0,
      evaluatedJobsCount: (json['evaluated_jobs_count'] as num?)?.toInt() ?? 0,
      evaluationCoveragePct: (json['evaluation_coverage_pct'] as num?)?.toDouble() ?? 0.0,
      overallAvgMatchScore: (json['overall_avg_match_score'] as num?)?.toDouble() ?? 0.0,
      remoteJobsCount: (json['remote_jobs_count'] as num?)?.toInt() ?? 0,
      remoteJobsPct: (json['remote_jobs_pct'] as num?)?.toDouble() ?? 0.0,
      topVolumeSource: json['top_volume_source'] as String? ?? 'N/A',
      topQualitySource: json['top_quality_source'] as String? ?? 'N/A',
    );
  }
}

/// Volume metrics for a single scraper source platform.
class SourceVolumeMetric {
  const SourceVolumeMetric({
    required this.source,
    required this.totalJobs,
    required this.jobsLast24h,
    required this.jobsLast7d,
    required this.remoteJobs,
    required this.onsiteJobs,
    required this.sharePct,
  });

  final String source;
  final int totalJobs;
  final int jobsLast24h;
  final int jobsLast7d;
  final int remoteJobs;
  final int onsiteJobs;
  final double sharePct;

  factory SourceVolumeMetric.fromJson(Map<String, dynamic> json) {
    return SourceVolumeMetric(
      source: json['source'] as String? ?? 'unknown',
      totalJobs: (json['total_jobs'] as num?)?.toInt() ?? 0,
      jobsLast24h: (json['jobs_last_24h'] as num?)?.toInt() ?? 0,
      jobsLast7d: (json['jobs_last_7d'] as num?)?.toInt() ?? 0,
      remoteJobs: (json['remote_jobs'] as num?)?.toInt() ?? 0,
      onsiteJobs: (json['onsite_jobs'] as num?)?.toInt() ?? 0,
      sharePct: (json['share_pct'] as num?)?.toDouble() ?? 0.0,
    );
  }
}

/// Quality and AI match yield metrics for a specific source platform.
class SourceQualityMetric {
  const SourceQualityMetric({
    required this.source,
    required this.evaluatedCount,
    required this.avgScore,
    required this.eliteMatches,
    required this.goodMatches,
    required this.lowMatches,
    required this.highMatchYieldPct,
  });

  final String source;
  final int evaluatedCount;
  final double avgScore;
  final int eliteMatches;
  final int goodMatches;
  final int lowMatches;
  final double highMatchYieldPct;

  factory SourceQualityMetric.fromJson(Map<String, dynamic> json) {
    return SourceQualityMetric(
      source: json['source'] as String? ?? 'unknown',
      evaluatedCount: (json['evaluated_count'] as num?)?.toInt() ?? 0,
      avgScore: (json['avg_score'] as num?)?.toDouble() ?? 0.0,
      eliteMatches: (json['elite_matches'] as num?)?.toInt() ?? 0,
      goodMatches: (json['good_matches'] as num?)?.toInt() ?? 0,
      lowMatches: (json['low_matches'] as num?)?.toInt() ?? 0,
      highMatchYieldPct: (json['high_match_yield_pct'] as num?)?.toDouble() ?? 0.0,
    );
  }
}

/// Day-level volume metric for the ingestion trend timeline.
class DailyIngestionMetric {
  const DailyIngestionMetric({
    required this.date,
    required this.jobsCount,
  });

  final String date;
  final int jobsCount;

  factory DailyIngestionMetric.fromJson(Map<String, dynamic> json) {
    return DailyIngestionMetric(
      date: json['date'] as String? ?? '',
      jobsCount: (json['jobs_count'] as num?)?.toInt() ?? 0,
    );
  }
}

/// Database-wide AI match score tier distribution.
class ScoreDistributionMetric {
  const ScoreDistributionMetric({
    required this.tier90To100,
    required this.tier80To89,
    required this.tier60To79,
    required this.tierBelow60,
    required this.unevaluatedCount,
    required this.avgScore,
  });

  final int tier90To100;
  final int tier80To89;
  final int tier60To79;
  final int tierBelow60;
  final int unevaluatedCount;
  final double avgScore;

  int get totalEvaluated => tier90To100 + tier80To89 + tier60To79 + tierBelow60;

  factory ScoreDistributionMetric.fromJson(Map<String, dynamic> json) {
    return ScoreDistributionMetric(
      tier90To100: (json['tier_90_100'] as num?)?.toInt() ?? 0,
      tier80To89: (json['tier_80_89'] as num?)?.toInt() ?? 0,
      tier60To79: (json['tier_60_79'] as num?)?.toInt() ?? 0,
      tierBelow60: (json['tier_below_60'] as num?)?.toInt() ?? 0,
      unevaluatedCount: (json['unevaluated_count'] as num?)?.toInt() ?? 0,
      avgScore: (json['avg_score'] as num?)?.toDouble() ?? 0.0,
    );
  }
}

/// Execution reliability and duration statistics for scraper runs.
class ScraperRunHealthMetric {
  const ScraperRunHealthMetric({
    required this.totalRunsRecorded,
    required this.successfulRuns,
    required this.failedRuns,
    required this.successRatePct,
    required this.avgDurationSeconds,
  });

  final int totalRunsRecorded;
  final int successfulRuns;
  final int failedRuns;
  final double successRatePct;
  final int avgDurationSeconds;

  factory ScraperRunHealthMetric.fromJson(Map<String, dynamic> json) {
    return ScraperRunHealthMetric(
      totalRunsRecorded: (json['total_runs_recorded'] as num?)?.toInt() ?? 0,
      successfulRuns: (json['successful_runs'] as num?)?.toInt() ?? 0,
      failedRuns: (json['failed_runs'] as num?)?.toInt() ?? 0,
      successRatePct: (json['success_rate_pct'] as num?)?.toDouble() ?? 0.0,
      avgDurationSeconds: (json['avg_duration_seconds'] as num?)?.toInt() ?? 0,
    );
  }
}

/// Representation of a top discovered hiring company.
class TopCompanyMetric {
  const TopCompanyMetric({
    required this.companyName,
    required this.jobCount,
  });

  final String companyName;
  final int jobCount;

  factory TopCompanyMetric.fromJson(Map<String, dynamic> json) {
    return TopCompanyMetric(
      companyName: json['company_name'] as String? ?? '',
      jobCount: (json['job_count'] as num?)?.toInt() ?? 0,
    );
  }
}

/// Representation of an individual scraper execution run.
class ScraperRunLog {
  const ScraperRunLog({
    required this.runId,
    required this.startedAt,
    required this.finishedAt,
    required this.status,
    required this.jobsAdded,
    required this.sourcesRaw,
    required this.errorMessage,
    required this.durationSeconds,
  });

  final String runId;
  final String startedAt;
  final String finishedAt;
  final String status;
  final int jobsAdded;
  final String sourcesRaw;
  final String errorMessage;
  final int durationSeconds;

  List<String> get sourcesList {
    if (sourcesRaw.isEmpty) return const [];
    try {
      final decoded = jsonDecode(sourcesRaw);
      if (decoded is List) {
        return decoded.map((e) => e.toString()).toList();
      }
    } catch (_) {}
    return const [];
  }

  factory ScraperRunLog.fromJson(Map<String, dynamic> json) {
    return ScraperRunLog(
      runId: json['run_id'] as String? ?? '',
      startedAt: json['started_at'] as String? ?? '',
      finishedAt: json['finished_at'] as String? ?? '',
      status: json['status'] as String? ?? 'completed',
      jobsAdded: (json['jobs_added'] as num?)?.toInt() ?? 0,
      sourcesRaw: json['sources_hit'] as String? ?? '[]',
      errorMessage: json['error_message'] as String? ?? '',
      durationSeconds: (json['duration_seconds'] as num?)?.toInt() ?? 0,
    );
  }
}

/// Complete telemetry dataset returned by the master admin stats endpoint.
class ScraperTelemetryData {
  const ScraperTelemetryData({
    required this.kpis,
    required this.sourcesVolume,
    required this.sourcesQuality,
    required this.ingestionTimeline,
    required this.scoreDistribution,
    required this.runHealth,
    required this.topCompanies,
    required this.runs,
  });

  final TelemetryKpis kpis;
  final List<SourceVolumeMetric> sourcesVolume;
  final List<SourceQualityMetric> sourcesQuality;
  final List<DailyIngestionMetric> ingestionTimeline;
  final ScoreDistributionMetric scoreDistribution;
  final ScraperRunHealthMetric runHealth;
  final List<TopCompanyMetric> topCompanies;
  final List<ScraperRunLog> runs;

  factory ScraperTelemetryData.fromJson(Map<String, dynamic> json) {
    final kpisMap = (json['kpis'] as Map<String, dynamic>?) ?? {};
    final sourcesVolumeList = (json['sources_volume'] as List?) ?? [];
    final sourcesQualityList = (json['sources_quality'] as List?) ?? [];
    final timelineList = (json['ingestion_timeline'] as List?) ?? [];
    final scoreDistMap = (json['score_distribution'] as Map<String, dynamic>?) ?? {};
    final runHealthMap = (json['run_health'] as Map<String, dynamic>?) ?? {};
    final topCompaniesList = (json['top_companies'] as List?) ?? [];
    final runsList = (json['runs'] as List?) ?? [];

    return ScraperTelemetryData(
      kpis: TelemetryKpis.fromJson(kpisMap.isNotEmpty ? kpisMap : json),
      sourcesVolume: sourcesVolumeList
          .map((item) => SourceVolumeMetric.fromJson(Map<String, dynamic>.from(item as Map)))
          .toList(),
      sourcesQuality: sourcesQualityList
          .map((item) => SourceQualityMetric.fromJson(Map<String, dynamic>.from(item as Map)))
          .toList(),
      ingestionTimeline: timelineList
          .map((item) => DailyIngestionMetric.fromJson(Map<String, dynamic>.from(item as Map)))
          .toList(),
      scoreDistribution: ScoreDistributionMetric.fromJson(scoreDistMap),
      runHealth: ScraperRunHealthMetric.fromJson(runHealthMap),
      topCompanies: topCompaniesList
          .map((item) => TopCompanyMetric.fromJson(Map<String, dynamic>.from(item as Map)))
          .toList(),
      runs: runsList
          .map((item) => ScraperRunLog.fromJson(Map<String, dynamic>.from(item as Map)))
          .toList(),
    );
  }
}
