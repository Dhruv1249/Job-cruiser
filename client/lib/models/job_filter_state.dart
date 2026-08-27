import 'dart:convert';
import 'package:shared_preferences/shared_preferences.dart';
import 'job.dart';

/// Immutable model representing the active filtering and sorting configuration for job listings.
class JobFilterState {
  const JobFilterState({
    this.matchScope = 'all',
    this.minScore = 0,
    this.maxScore = 100,
    this.recencyDays,
    this.viewMode = 'all',
    this.workModel = 'all',
    this.applicationStatus = 'all',
    this.sortBy = 'score_desc',
    this.searchQuery = '',
  });

  static const String storageKey = 'jobcruiser_filter_state_v1';

  final String matchScope;
  final int minScore;
  final int maxScore;
  final int? recencyDays;
  final String viewMode;
  final String workModel;
  final String applicationStatus;
  final String sortBy;
  final String searchQuery;

  /// Returns true if all filter dimensions match their default values.
  bool get isDefault {
    return matchScope == 'all' &&
        minScore == 0 &&
        maxScore == 100 &&
        recencyDays == null &&
        viewMode == 'all' &&
        workModel == 'all' &&
        applicationStatus == 'all' &&
        sortBy == 'score_desc' &&
        searchQuery.isEmpty;
  }

  /// Calculates the number of non-default filter settings currently active.
  int get activeFilterCount {
    var count = 0;
    if (matchScope != 'all') count++;
    if (minScore > 0 || maxScore < 100) count++;
    if (recencyDays != null && recencyDays! > 0) count++;
    if (viewMode != 'all') count++;
    if (workModel != 'all') count++;
    if (applicationStatus != 'all') count++;
    if (sortBy != 'score_desc') count++;
    if (searchQuery.trim().isNotEmpty) count++;
    return count;
  }

  /// Returns a clean default instance of [JobFilterState].
  JobFilterState reset() {
    return const JobFilterState();
  }

  /// Creates a copy of [JobFilterState] with selectively replaced fields.
  JobFilterState copyWith({
    String? matchScope,
    int? minScore,
    int? maxScore,
    int? Function()? recencyDays,
    String? viewMode,
    String? workModel,
    String? applicationStatus,
    String? sortBy,
    String? searchQuery,
  }) {
    return JobFilterState(
      matchScope: matchScope ?? this.matchScope,
      minScore: minScore ?? this.minScore,
      maxScore: maxScore ?? this.maxScore,
      recencyDays: recencyDays != null ? recencyDays() : this.recencyDays,
      viewMode: viewMode ?? this.viewMode,
      workModel: workModel ?? this.workModel,
      applicationStatus: applicationStatus ?? this.applicationStatus,
      sortBy: sortBy ?? this.sortBy,
      searchQuery: searchQuery ?? this.searchQuery,
    );
  }

  /// Converts the current state to a JSON-encodable map.
  Map<String, dynamic> toJson() {
    return {
      'match_scope': matchScope,
      'min_score': minScore,
      'max_score': maxScore,
      'recency_days': recencyDays,
      'view_mode': viewMode,
      'work_model': workModel,
      'application_status': applicationStatus,
      'sort_by': sortBy,
      'search_query': searchQuery,
    };
  }

  /// Constructs a [JobFilterState] from a JSON map.
  factory JobFilterState.fromJson(Map<String, dynamic> json) {
    return JobFilterState(
      matchScope: json['match_scope'] as String? ?? 'all',
      minScore: (json['min_score'] as num?)?.toInt() ?? 0,
      maxScore: (json['max_score'] as num?)?.toInt() ?? 100,
      recencyDays: (json['recency_days'] as num?)?.toInt(),
      viewMode: json['view_mode'] as String? ?? 'all',
      workModel: json['work_model'] as String? ?? 'all',
      applicationStatus: json['application_status'] as String? ?? 'all',
      sortBy: json['sort_by'] as String? ?? 'score_desc',
      searchQuery: json['search_query'] as String? ?? '',
    );
  }

  /// Loads saved filter preferences from persistent local storage.
  static Future<JobFilterState> loadFromStorage() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final raw = prefs.getString(storageKey);
      if (raw != null && raw.isNotEmpty) {
        final decoded = jsonDecode(raw) as Map<String, dynamic>;
        return JobFilterState.fromJson(decoded);
      }
    } catch (_) {}
    return const JobFilterState();
  }

  /// Persists current filter settings to local device storage.
  Future<void> saveToStorage() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(storageKey, jsonEncode(toJson()));
    } catch (_) {}
  }

  /// Evaluates whether a given [MatchedJob] satisfies all active filter rules.
  bool matchesJob(MatchedJob job) {
    if (matchScope == 'matched_only' && !job.isMatched && job.matchScore <= 0) {
      return false;
    }
    if (matchScope == 'unmatched_only' && (job.isMatched || job.matchScore > 0)) {
      return false;
    }

    if (job.matchScore < minScore || job.matchScore > maxScore) {
      return false;
    }

    if (viewMode == 'unviewed' && job.isViewed) {
      return false;
    }
    if (viewMode == 'viewed' && !job.isViewed) {
      return false;
    }

    if (workModel == 'remote_only' && !job.isRemote) {
      return false;
    }
    if (workModel == 'onsite_hybrid' && job.isRemote) {
      return false;
    }

    if (applicationStatus != 'all') {
      if (applicationStatus == 'unapplied' && job.applicationStatus != 'unapplied') {
        return false;
      } else if (applicationStatus != 'unapplied' &&
          job.applicationStatus.toLowerCase() != applicationStatus.toLowerCase()) {
        return false;
      }
    }

    if (recencyDays != null && recencyDays! > 0) {
      final targetDate = _extractJobDate(job);
      if (targetDate != null) {
        final cutoff = DateTime.now().subtract(Duration(days: recencyDays!));
        if (targetDate.isBefore(cutoff)) {
          return false;
        }
      }
    }

    if (searchQuery.trim().isNotEmpty) {
      final query = searchQuery.trim().toLowerCase();
      final titleMatch = job.title.toLowerCase().contains(query);
      final companyMatch = job.company.toLowerCase().contains(query);
      final locationMatch = job.location.toLowerCase().contains(query);
      final summaryMatch = job.summary.toLowerCase().contains(query);
      final reasoningMatch = job.matchReasoning.toLowerCase().contains(query);
      final techMatch = job.techStack.any(
        (tech) => tech.toLowerCase().contains(query),
      );
      if (!titleMatch &&
          !companyMatch &&
          !locationMatch &&
          !summaryMatch &&
          !reasoningMatch &&
          !techMatch) {
        return false;
      }
    }

    return true;
  }

  /// Sorts a list of [MatchedJob] items according to the active sorting strategy.
  List<MatchedJob> applySort(List<MatchedJob> jobs) {
    final list = List<MatchedJob>.from(jobs);
    switch (sortBy) {
      case 'date_desc':
        list.sort((first, second) {
          final firstDate = _extractJobDate(first) ?? DateTime.fromMillisecondsSinceEpoch(0);
          final secondDate = _extractJobDate(second) ?? DateTime.fromMillisecondsSinceEpoch(0);
          return secondDate.compareTo(firstDate);
        });
        break;
      case 'date_asc':
        list.sort((first, second) {
          final firstDate = _extractJobDate(first) ?? DateTime.fromMillisecondsSinceEpoch(0);
          final secondDate = _extractJobDate(second) ?? DateTime.fromMillisecondsSinceEpoch(0);
          return firstDate.compareTo(secondDate);
        });
        break;
      case 'salary_desc':
        list.sort((first, second) {
          final firstSalary = first.salaryMax ?? first.salaryMin ?? 0;
          final secondSalary = second.salaryMax ?? second.salaryMin ?? 0;
          if (secondSalary != firstSalary) {
            return secondSalary.compareTo(firstSalary);
          }
          return second.matchScore.compareTo(first.matchScore);
        });
        break;
      case 'score_desc':
      default:
        list.sort((first, second) {
          if (second.matchScore != first.matchScore) {
            return second.matchScore.compareTo(first.matchScore);
          }
          if (first.isViewed != second.isViewed) {
            return first.isViewed ? 1 : -1;
          }
          final firstDate = _extractJobDate(first) ?? DateTime.fromMillisecondsSinceEpoch(0);
          final secondDate = _extractJobDate(second) ?? DateTime.fromMillisecondsSinceEpoch(0);
          return secondDate.compareTo(firstDate);
        });
        break;
    }
    return list;
  }

  DateTime? _extractJobDate(MatchedJob job) {
    if (job.scrapedAt.isNotEmpty) {
      final parsed = DateTime.tryParse(job.scrapedAt.replaceAll(' ', 'T'));
      if (parsed != null) return parsed;
    }
    if (job.postedDate.isNotEmpty) {
      final parsed = DateTime.tryParse(job.postedDate.replaceAll(' ', 'T'));
      if (parsed != null) return parsed;
    }
    return null;
  }
}
