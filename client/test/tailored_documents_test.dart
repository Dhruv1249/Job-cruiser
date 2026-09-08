import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_app/screens/tailored_documents_screen.dart';

void main() {
  group('TailoredJobDocumentGroup Logic & Widget Tests', () {
    final mockResumes = [
      {
        'id': 'res-1',
        'job_id': 'job-abc',
        'label': 'Acme Corp — Senior Backend Engineer (2026-09-08)',
        'overleaf_folder_path': 'job_applications/AcmeCorp_SeniorBackendEngineer_20260908',
        'page_count': 1,
        'status': 'ready',
        'created_at': '2026-09-08T10:00:00Z',
        'is_default': false,
      },
      {
        'id': 'res-2',
        'job_id': null,
        'label': 'General Master CV',
        'overleaf_folder_path': '',
        'page_count': 1,
        'status': 'ready',
        'created_at': '2026-09-01T10:00:00Z',
        'is_default': true,
      },
    ];

    final mockCoverLetters = [
      {
        'id': 'cov-1',
        'job_id': 'job-abc',
        'label': 'Acme Corp — Senior Backend Engineer (Cover Letter 2026-09-08)',
        'overleaf_folder_path': 'job_applications/AcmeCorp_SeniorBackendEngineer_20260908',
        'page_count': 1,
        'status': 'ready',
        'created_at': '2026-09-08T10:00:00Z',
      },
    ];

    test('groupTailoredDocuments pairs resume and cover letter by job', () {
      final groups = groupTailoredDocuments(mockResumes, mockCoverLetters);

      expect(groups.length, equals(2));

      final acmeGroup = groups.firstWhere((g) => g.company == 'Acme Corp');
      expect(acmeGroup.role, equals('Senior Backend Engineer'));
      expect(acmeGroup.resumeVersion, isNotNull);
      expect(acmeGroup.coverLetterVersion, isNotNull);
      expect(acmeGroup.resumeVersion!['id'], equals('res-1'));
      expect(acmeGroup.coverLetterVersion!['id'], equals('cov-1'));

      final generalGroup = groups.firstWhere((g) => g.company != 'Acme Corp');
      expect(generalGroup.resumeVersion, isNotNull);
      expect(generalGroup.coverLetterVersion, isNull);
    });

    testWidgets('TailoredJobCard displays job header and expands to reveal CV and Cover Letter', (tester) async {
      final groups = groupTailoredDocuments(mockResumes, mockCoverLetters);
      final acmeGroup = groups.firstWhere((g) => g.company == 'Acme Corp');

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SingleChildScrollView(
              child: TailoredJobCard(
                group: acmeGroup,
                onViewDocument: (doc, type) {},
                onDeleteDocument: (docId, type) {},
                onSetDefaultResume: (docId) {},
              ),
            ),
          ),
        ),
      );

      expect(find.text('Senior Backend Engineer'), findsOneWidget);
      expect(find.textContaining('Acme Corp'), findsOneWidget);
      expect(find.text('Tailored CV / Resume'), findsNothing);
      expect(find.text('Generated Cover Letter'), findsNothing);

      await tester.tap(find.text('Senior Backend Engineer'));
      await tester.pumpAndSettle();

      expect(find.text('Tailored CV / Resume'), findsOneWidget);
      expect(find.text('Generated Cover Letter'), findsOneWidget);
      expect(find.text('View PDF'), findsNWidgets(2));
    });
  });
}
