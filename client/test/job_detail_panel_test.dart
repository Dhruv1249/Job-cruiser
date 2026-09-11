import 'package:flutter/material.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_app/models/job.dart';
import 'package:flutter_app/widgets/job_detail_panel.dart';

void main() {
  setUp(() {
    dotenv.loadFromString(envString: 'API_BASE_URL=http://localhost:8080');
  });

  group('JobDetailPanel Responsive Action Bar Tests', () {
    const sampleJob = MatchedJob(
      jobId: 'job-test-1',
      title: 'Senior Flutter Developer',
      company: 'Antigravity Systems',
      location: 'Remote (India)',
      isRemote: true,
      url: 'https://example.com/jobs/flutter-1',
      postedDate: '2026-09-08',
      source: 'indeed',
      matchScore: 92,
      matchReasoning: 'Matches candidate profile with deep Dart and Flutter knowledge.',
      summary: 'Matches candidate profile perfectly with mobile development experience.',
      seniority: 'Senior',
      techStack: ['Flutter', 'Dart', 'Go'],
      rawDescription: 'Looking for a senior Flutter developer to build desktop and mobile apps.',
      isMatched: true,
      applicationStatus: 'unapplied',
      isViewed: false,
    );

    testWidgets('renders descriptive labeled buttons and tailoring info on wide screens', (tester) async {
      await tester.binding.setSurfaceSize(const Size(1024, 768));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: JobDetailPanel(
              job: sampleJob,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Save'), findsOneWidget);
      expect(find.text('Mark Applied'), findsOneWidget);
      expect(find.text('Apply on ATS'), findsOneWidget);
      expect(find.text('Hide'), findsOneWidget);
      expect(find.text('Tailor Application'), findsOneWidget);
      expect(find.text('ATS CV & Cover Letter in Overleaf'), findsOneWidget);
    });

    testWidgets('renders compact icon buttons on narrow mobile screens', (tester) async {
      await tester.binding.setSurfaceSize(const Size(380, 700));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: JobDetailPanel(
              job: sampleJob,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Save'), findsNothing);
      expect(find.text('Mark Applied'), findsNothing);
      expect(find.text('Apply on ATS'), findsNothing);
      expect(find.text('Tailor Application'), findsOneWidget);
      expect(find.byType(IconButton), findsWidgets);
    });

    testWidgets('renders Open Tailored Documents and banner when job has tailored docs on wide screens', (tester) async {
      await tester.binding.setSurfaceSize(const Size(1024, 768));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      final tailoredJob = sampleJob.copyWith(hasTailoredDocs: true);

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: JobDetailPanel(
              job: tailoredJob,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Tailored Documents Generated'), findsOneWidget);
      expect(find.text('CV & Cover Letter Ready'), findsOneWidget);
      expect(find.text('Open Tailored Documents'), findsOneWidget);
      expect(find.text('Tailor Application'), findsNothing);
    });

    testWidgets('renders Open Tailored Documents button on narrow mobile screens when tailored', (tester) async {
      await tester.binding.setSurfaceSize(const Size(380, 700));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      final tailoredJob = sampleJob.copyWith(hasTailoredDocs: true);

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: JobDetailPanel(
              job: tailoredJob,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Open Tailored Documents'), findsOneWidget);
      expect(find.text('Tailor Application'), findsNothing);
    });
  });
}
