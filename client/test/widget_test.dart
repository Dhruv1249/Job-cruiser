import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_app/models/job.dart';

void main() {
  test('MatchedJob model parses isViewed and applicationStatus correctly', () {
    final json = {
      'job_id': 'job-123',
      'title': 'Go Backend Engineer',
      'company': 'Tech Corp',
      'match_score': 95,
      'is_viewed': true,
      'application_status': 'applied',
    };

    final job = MatchedJob.fromJson(json);
    expect(job.jobId, equals('job-123'));
    expect(job.matchScore, equals(95));
    expect(job.isViewed, isTrue);
    expect(job.applicationStatus, equals('applied'));
  });

  test('MatchedJob model defaults isViewed to false and applicationStatus to unapplied', () {
    final json = {
      'job_id': 'job-456',
      'title': 'Flutter Developer',
      'company': 'Mobile Inc',
    };

    final job = MatchedJob.fromJson(json);
    expect(job.isViewed, isFalse);
    expect(job.applicationStatus, equals('unapplied'));
  });
}
