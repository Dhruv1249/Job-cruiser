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
    this.scrapedAt = '',
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
    this.salaryPeriod,
    this.employmentType,
    this.isViewed = false,
    this.applicationStatus = 'unapplied',
    this.isNew = false,
    this.hasTailoredDocs = false,
  });

  final String jobId;
  final String title;
  final String company;
  final String location;
  final bool isRemote;
  final String source;
  final String url;
  final String postedDate;
  final String scrapedAt;
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
  final String? salaryPeriod;
  final String? employmentType;
  final bool isViewed;
  final String applicationStatus;
  final bool isNew;
  final bool hasTailoredDocs;

  /// Creates a copy of [MatchedJob] with overridden fields.
  MatchedJob copyWith({
    bool? isViewed,
    String? applicationStatus,
    bool? isNew,
    bool? hasTailoredDocs,
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
      scrapedAt: scrapedAt,
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
      salaryPeriod: salaryPeriod,
      employmentType: employmentType,
      isViewed: isViewed ?? this.isViewed,
      applicationStatus: applicationStatus ?? this.applicationStatus,
      isNew: isNew ?? this.isNew,
      hasTailoredDocs: hasTailoredDocs ?? this.hasTailoredDocs,
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
      scrapedAt: json['scraped_at'] as String? ?? '',
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
      salaryPeriod: json['salary_period'] as String?,
      employmentType: json['employment_type'] as String? ?? json['job_type'] as String?,
      isViewed: isViewedVal,
      applicationStatus: json['application_status'] as String? ?? 'unapplied',
      isNew: json['is_new'] as bool? ?? (!isViewedVal),
      hasTailoredDocs: json['has_tailored_docs'] as bool? ?? false,
    );
  }

  /// Formatted salary string representation.
  String get salaryText {
    final minVal = salaryMin;
    final maxVal = salaryMax;

    if ((minVal == null || minVal <= 0) && (maxVal == null || maxVal <= 0)) {
      return '';
    }

    final cleanCurrency = currency.toUpperCase().trim();
    String symbol = '\$';
    if (cleanCurrency == 'INR') {
      symbol = '₹';
    } else if (cleanCurrency == 'EUR') {
      symbol = '€';
    } else if (cleanCurrency == 'GBP') {
      symbol = '£';
    } else if (cleanCurrency == 'CAD') {
      symbol = 'CA\$';
    } else if (cleanCurrency == 'AUD') {
      symbol = 'A\$';
    } else if (cleanCurrency == 'SGD') {
      symbol = 'S\$';
    } else if (cleanCurrency.isNotEmpty && cleanCurrency != 'USD') {
      symbol = '$cleanCurrency ';
    }

    final cleanPeriod = salaryPeriod?.toLowerCase().trim() ?? '';
    final isStipendOrMonthly = cleanPeriod == 'monthly' || cleanPeriod == 'stipend';
    final isHourly = cleanPeriod == 'hourly';
    final isWeekly = cleanPeriod == 'weekly';

    String formatSingleAmount(int amount) {
      if (cleanCurrency == 'INR') {
        if (!isStipendOrMonthly && amount >= 100000) {
          final lpa = amount / 100000;
          final formattedLpa = lpa % 1 == 0 ? lpa.toInt().toString() : lpa.toStringAsFixed(1);
          return '$symbol$formattedLpa LPA';
        }
        if (amount >= 1000) {
          final kVal = amount / 1000;
          final formattedK = kVal % 1 == 0 ? kVal.toInt().toString() : kVal.toStringAsFixed(1);
          return '$symbol${formattedK}k';
        }
        return '$symbol$amount';
      }

      if (isHourly) {
        return '$symbol$amount';
      }

      if (amount >= 1000) {
        final kVal = amount / 1000;
        final formattedK = kVal % 1 == 0 ? kVal.toInt().toString() : kVal.toStringAsFixed(1);
        return '$symbol${formattedK}k';
      }
      return '$symbol$amount';
    }

    String periodSuffix = '';
    if (isHourly) {
      periodSuffix = '/hr';
    } else if (cleanPeriod == 'stipend') {
      periodSuffix = '/stipend';
    } else if (isStipendOrMonthly) {
      periodSuffix = '/mo';
    } else if (isWeekly) {
      periodSuffix = '/wk';
    }

    if (minVal != null && minVal > 0 && maxVal != null && maxVal > 0) {
      if (minVal == maxVal) {
        final formatted = formatSingleAmount(minVal);
        if (formatted.endsWith('LPA') || periodSuffix.isEmpty) {
          return formatted;
        }
        return '$formatted$periodSuffix';
      }

      final formattedMin = formatSingleAmount(minVal);
      final formattedMax = formatSingleAmount(maxVal);

      if (formattedMin.endsWith('LPA') && formattedMax.endsWith('LPA')) {
        final minNum = formattedMin.replaceAll(' LPA', '');
        return '$minNum - $formattedMax';
      }
      if (periodSuffix.isNotEmpty) {
        return '$formattedMin - $formattedMax$periodSuffix';
      }
      return '$formattedMin - $formattedMax';
    } else if (minVal != null && minVal > 0) {
      final formatted = formatSingleAmount(minVal);
      if (formatted.endsWith('LPA') || periodSuffix.isEmpty) {
        return '$formatted+';
      }
      return '$formatted+$periodSuffix';
    } else if (maxVal != null && maxVal > 0) {
      final formatted = formatSingleAmount(maxVal);
      if (formatted.endsWith('LPA') || periodSuffix.isEmpty) {
        return 'Up to $formatted';
      }
      return 'Up to $formatted$periodSuffix';
    }

    return '';
  }

  /// Formatted employment type badge text.
  String get employmentTypeDisplay {
    final type = employmentType?.toLowerCase().trim() ?? '';
    if (type == 'intern_ppo' || type.contains('ppo')) {
      return 'Intern + PPO';
    }
    if (type == 'intern' || type == 'internship') {
      return 'Internship';
    }
    if (type == 'contract') {
      return 'Contract';
    }
    if (type == 'freelance') {
      return 'Freelance';
    }
    if (type == 'full_time' || type == 'full-time' || type == 'fulltime') {
      return 'Full-Time';
    }

    final lowerTitle = title.toLowerCase();
    final lowerSeniority = seniority.toLowerCase();
    if (lowerTitle.contains('ppo') || lowerSeniority.contains('ppo')) {
      return 'Intern + PPO';
    }
    if (lowerTitle.contains('intern') || lowerSeniority.contains('intern')) {
      return 'Internship';
    }
    if (lowerTitle.contains('contract') || lowerSeniority.contains('contract')) {
      return 'Contract';
    }
    if (lowerTitle.contains('freelance')) {
      return 'Freelance';
    }
    return '';
  }

  /// Indicates if employment type is a non-standard track like intern, contract, or freelance.
  bool get isSpecialEmploymentType {
    final display = employmentTypeDisplay;
    return display.isNotEmpty && display != 'Full-Time';
  }

  /// Relative human-readable string indicating when the job listing was scraped.
  String get scrapedAgoText {
    if (scrapedAt.isEmpty) return '';
    try {
      final parsedDate = DateTime.tryParse(scrapedAt.replaceAll(' ', 'T'));
      if (parsedDate == null) return scrapedAt;

      final diff = DateTime.now().difference(parsedDate);
      if (diff.inMinutes < 60) {
        final mins = diff.inMinutes <= 0 ? 1 : diff.inMinutes;
        return '$mins m ago';
      } else if (diff.inHours < 24) {
        return '${diff.inHours}h ago';
      } else if (diff.inDays == 1) {
        return 'Yesterday';
      } else {
        return '${diff.inDays}d ago';
      }
    } catch (_) {
      return scrapedAt;
    }
  }
}
