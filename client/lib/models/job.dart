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
