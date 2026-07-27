import 'dart:convert';

/// Model class representing a matched job opportunity returned from the backend.
class MatchedJob {
  const MatchedJob({
    required this.jobId,
    required this.title,
    required this.company,
    required this.location,
    required this.isRemote,
    required this.source,
    required this.url,
    required this.postedDate,
    required this.seniority,
    required this.summary,
    this.rawDescription = '',
    required this.matchScore,
    required this.matchReasoning,
    required this.techStack,
    required this.isMatched,
    this.salaryMin,
    this.salaryMax,
    this.currency = 'USD',
    this.isViewed = false,
    this.applicationStatus = 'unapplied',
    this.isNew = false,
  });

  final String jobId;
  final String title;
  final String company;
  final String location;
  final bool isRemote;
  final String source;
  final String url;
  final String postedDate;
  final String seniority;
  final String summary;
  final String rawDescription;
  final int matchScore;
  final String matchReasoning;
  final List<String> techStack;
  final bool isMatched;
  final int? salaryMin;
  final int? salaryMax;
  final String currency;
  final bool isViewed;
  final String applicationStatus;
  final bool isNew;

  /// Creates a copy of [MatchedJob] with overridden fields.
  MatchedJob copyWith({
    bool? isViewed,
    String? applicationStatus,
    bool? isNew,
  }) {
    return MatchedJob(
      jobId: jobId,
      title: title,
      company: company,
      location: location,
      isRemote: isRemote,
      source: source,
      url: url,
      postedDate: postedDate,
      seniority: seniority,
      summary: summary,
      rawDescription: rawDescription,
      matchScore: matchScore,
      matchReasoning: matchReasoning,
      techStack: techStack,
      isMatched: isMatched,
      salaryMin: salaryMin,
      salaryMax: salaryMax,
      currency: currency,
      isViewed: isViewed ?? this.isViewed,
      applicationStatus: applicationStatus ?? this.applicationStatus,
      isNew: isNew ?? this.isNew,
    );
  }

  /// Creates a [MatchedJob] instance from a JSON map response.
  factory MatchedJob.fromJson(Map<String, dynamic> json) {
    List<String> parsedTechStack = [];
    if (json['tech_stack'] != null) {
      if (json['tech_stack'] is List) {
        parsedTechStack = List<String>.from(
          (json['tech_stack'] as List).map((item) => item.toString()),
        );
      } else if (json['tech_stack'] is String) {
        try {
          final decoded = jsonDecode(json['tech_stack'] as String);
          if (decoded is List) {
            parsedTechStack = List<String>.from(
              decoded.map((item) => item.toString()),
            );
          }
        } catch (_) {}
      }
    }

    final isViewedVal = json['is_viewed'] as bool? ?? false;

    return MatchedJob(
      jobId: json['job_id'] as String? ?? json['id'] as String? ?? '',
      title: json['title'] as String? ?? 'Untitled Position',
      company: json['company'] as String? ?? 'Unknown Company',
      location: json['location'] as String? ?? 'Remote',
      isRemote: json['is_remote'] as bool? ?? false,
      source: json['source'] as String? ?? '',
      url: json['url'] as String? ?? '',
      postedDate: json['posted_date'] as String? ?? '',
      seniority: json['seniority'] as String? ?? '',
      summary: json['summary'] as String? ?? '',
      rawDescription: json['raw_description'] as String? ?? json['raw_desc'] as String? ?? '',
      matchScore: (json['match_score'] as num?)?.toInt() ?? 0,
      matchReasoning: json['match_reasoning'] as String? ?? '',
      techStack: parsedTechStack,
      isMatched: json['is_matched'] as bool? ?? false,
      salaryMin: json['salary_min'] as int?,
      salaryMax: json['salary_max'] as int?,
      currency: json['currency'] as String? ?? 'USD',
      isViewed: isViewedVal,
      applicationStatus: json['application_status'] as String? ?? 'unapplied',
      isNew: json['is_new'] as bool? ?? (!isViewedVal),
    );
  }

  /// Formatted salary string representation.
  String get salaryText {
    if (salaryMin != null && salaryMax != null) {
      return '\$$salaryMin - \$$salaryMax';
    } else if (salaryMin != null) {
      return '\$$salaryMin+';
    } else if (salaryMax != null) {
      return 'Up to \$$salaryMax';
    }
    return '';
  }
}
