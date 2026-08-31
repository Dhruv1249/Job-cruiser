import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_app/models/job_filter_state.dart';
import 'package:flutter_app/models/job.dart';

void main() {
  group('JobFilterState Model & Serialization', () {
    test('default state has expected initial properties', () {
      const state = JobFilterState();
      expect(state.matchScope, equals('all'));
      expect(state.minScore, equals(0));
      expect(state.maxScore, equals(100));
      expect(state.recencyDays, isNull);
      expect(state.viewMode, equals('all'));
      expect(state.workModel, equals('all'));
      expect(state.applicationStatus, equals('all'));
      expect(state.sortBy, equals('score_desc'));
      expect(state.searchQuery, isEmpty);
      expect(state.isDefault, isTrue);
      expect(state.activeFilterCount, equals(0));
    });

    test('serializes to JSON and restores from JSON accurately', () {
      const state = JobFilterState(
        matchScope: 'matched_only',
        minScore: 70,
        maxScore: 95,
        recencyDays: 3,
        viewMode: 'unviewed',
        workModel: 'remote_only',
        applicationStatus: 'unapplied',
        sortBy: 'date_desc',
        searchQuery: 'golang',
      );

      final json = state.toJson();
      final restored = JobFilterState.fromJson(json);

      expect(restored.matchScope, equals('matched_only'));
      expect(restored.minScore, equals(70));
      expect(restored.maxScore, equals(95));
      expect(restored.recencyDays, equals(3));
      expect(restored.viewMode, equals('unviewed'));
      expect(restored.workModel, equals('remote_only'));
      expect(restored.applicationStatus, equals('unapplied'));
      expect(restored.sortBy, equals('date_desc'));
      expect(restored.searchQuery, equals('golang'));
      expect(restored.isDefault, isFalse);
      expect(restored.activeFilterCount, equals(8));
    });

    test('reset returns pure default filter state', () {
      const customized = JobFilterState(
        matchScope: 'unmatched_only',
        minScore: 50,
        recencyDays: 2,
        sortBy: 'date_desc',
      );
      final resetState = customized.reset();
      expect(resetState.isDefault, isTrue);
      expect(resetState.activeFilterCount, equals(0));
    });
  });

  group('JobFilterState Match Filtering', () {
    final now = DateTime.now();
    final oneDayAgoIso = now.subtract(const Duration(hours: 12)).toIso8601String();
    final fiveDaysAgoIso = now.subtract(const Duration(days: 5)).toIso8601String();

    final highMatchedJob = MatchedJob(
      jobId: 'job-1',
      title: 'Senior Backend Engineer',
      company: 'Alpha Tech',
      location: 'Remote',
      isRemote: true,
      source: 'Greenhouse',
      url: 'https://example.com/job-1',
      postedDate: '2026-08-25',
      scrapedAt: oneDayAgoIso,
      seniority: 'Senior',
      summary: 'Go and distributed systems role',
      rawDescription: 'Building high throughput Go microservices.',
      matchScore: 92,
      matchReasoning: 'Strong fit for Go backend',
      techStack: const ['Go', 'Postgres', 'Docker'],
      isMatched: true,
      salaryMin: 150000,
      salaryMax: 180000,
      isViewed: false,
      applicationStatus: 'unapplied',
    );

    final unmatchedJob = MatchedJob(
      jobId: 'job-2',
      title: 'Frontend React Developer',
      company: 'Beta Systems',
      location: 'New York, NY',
      isRemote: false,
      source: 'Lever',
      url: 'https://example.com/job-2',
      postedDate: '2026-08-20',
      scrapedAt: fiveDaysAgoIso,
      seniority: 'Mid',
      summary: 'React and UI engineer',
      rawDescription: 'Crafting responsive user interfaces with TypeScript.',
      matchScore: 0,
      matchReasoning: '',
      techStack: const ['React', 'TypeScript'],
      isMatched: false,
      salaryMin: 120000,
      salaryMax: 140000,
      isViewed: true,
      applicationStatus: 'bookmarked',
    );

    test('filters by match scope correctly', () {
      const allScope = JobFilterState(matchScope: 'all');
      expect(allScope.matchesJob(highMatchedJob), isTrue);
      expect(allScope.matchesJob(unmatchedJob), isTrue);

      const matchedOnly = JobFilterState(matchScope: 'matched_only');
      expect(matchedOnly.matchesJob(highMatchedJob), isTrue);
      expect(matchedOnly.matchesJob(unmatchedJob), isFalse);

      const unmatchedOnly = JobFilterState(matchScope: 'unmatched_only');
      expect(unmatchedOnly.matchesJob(highMatchedJob), isFalse);
      expect(unmatchedOnly.matchesJob(unmatchedJob), isTrue);
    });

    test('filters by match score range correctly', () {
      const highRange = JobFilterState(minScore: 80, maxScore: 100);
      expect(highRange.matchesJob(highMatchedJob), isTrue);
      expect(highRange.matchesJob(unmatchedJob), isFalse);

      const midRange = JobFilterState(minScore: 50, maxScore: 80);
      expect(midRange.matchesJob(highMatchedJob), isFalse);
      expect(midRange.matchesJob(unmatchedJob), isFalse);
    });

    test('filters by recency in days correctly', () {
      const past24Hours = JobFilterState(recencyDays: 1);
      expect(past24Hours.matchesJob(highMatchedJob), isTrue);
      expect(past24Hours.matchesJob(unmatchedJob), isFalse);

      const past7Days = JobFilterState(recencyDays: 7);
      expect(past7Days.matchesJob(highMatchedJob), isTrue);
      expect(past7Days.matchesJob(unmatchedJob), isTrue);
    });

    test('filters by remote work model correctly', () {
      const remoteOnly = JobFilterState(workModel: 'remote_only');
      expect(remoteOnly.matchesJob(highMatchedJob), isTrue);
      expect(remoteOnly.matchesJob(unmatchedJob), isFalse);

      const onsiteOnly = JobFilterState(workModel: 'onsite_hybrid');
      expect(onsiteOnly.matchesJob(highMatchedJob), isFalse);
      expect(onsiteOnly.matchesJob(unmatchedJob), isTrue);
    });

    test('filters by view mode correctly', () {
      const unviewedOnly = JobFilterState(viewMode: 'unviewed');
      expect(unviewedOnly.matchesJob(highMatchedJob), isTrue);
      expect(unviewedOnly.matchesJob(unmatchedJob), isFalse);

      const viewedOnly = JobFilterState(viewMode: 'viewed');
      expect(viewedOnly.matchesJob(highMatchedJob), isFalse);
      expect(viewedOnly.matchesJob(unmatchedJob), isTrue);
    });

    test('filters by search query across multiple fields', () {
      const titleSearch = JobFilterState(searchQuery: 'backend');
      expect(titleSearch.matchesJob(highMatchedJob), isTrue);
      expect(titleSearch.matchesJob(unmatchedJob), isFalse);

      const companySearch = JobFilterState(searchQuery: 'Beta');
      expect(companySearch.matchesJob(highMatchedJob), isFalse);
      expect(companySearch.matchesJob(unmatchedJob), isTrue);

      const techSearch = JobFilterState(searchQuery: 'docker');
      expect(techSearch.matchesJob(highMatchedJob), isTrue);
      expect(techSearch.matchesJob(unmatchedJob), isFalse);
    });

    test('filters by application status correctly', () {
      const unappliedFilter = JobFilterState(applicationStatus: 'unapplied');
      expect(unappliedFilter.matchesJob(highMatchedJob), isTrue);
      expect(unappliedFilter.matchesJob(unmatchedJob), isFalse);

      const bookmarkedFilter = JobFilterState(applicationStatus: 'bookmarked');
      expect(bookmarkedFilter.matchesJob(highMatchedJob), isFalse);
      expect(bookmarkedFilter.matchesJob(unmatchedJob), isTrue);

      const allStatusFilter = JobFilterState(applicationStatus: 'all');
      expect(allStatusFilter.matchesJob(highMatchedJob), isTrue);
      expect(allStatusFilter.matchesJob(unmatchedJob), isTrue);
    });

    test('combined filters apply all dimensions simultaneously', () {
      const strictFilter = JobFilterState(
        matchScope: 'matched_only',
        minScore: 80,
        workModel: 'remote_only',
        viewMode: 'unviewed',
        applicationStatus: 'unapplied',
      );
      expect(strictFilter.matchesJob(highMatchedJob), isTrue);
      expect(strictFilter.matchesJob(unmatchedJob), isFalse);
    });
  });
}
