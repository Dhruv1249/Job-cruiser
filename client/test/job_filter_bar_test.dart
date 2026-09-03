import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_app/models/job_filter_state.dart';
import 'package:flutter_app/widgets/job_filter_bar.dart';

void main() {
  group('JobFilterBar widget tests', () {
    testWidgets('renders all primary filter chips', (tester) async {
      const defaultState = JobFilterState();
      var dialogOpened = false;

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: JobFilterBar(
              filterState: defaultState,
              onFilterChanged: (_) {},
              onOpenFilterDialog: () => dialogOpened = true,
            ),
          ),
        ),
      );

      expect(find.text('All Filters'), findsOneWidget);
      expect(find.text('Sort: Score ↓'), findsOneWidget);
      expect(find.text('All Jobs'), findsOneWidget);
      expect(find.text('All Scores'), findsOneWidget);

      await tester.tap(find.text('All Filters'));
      await tester.pump();
      expect(dialogOpened, isTrue);
    });

    testWidgets('renders active count and reset chip when filter is modified', (tester) async {
      final modifiedState = const JobFilterState().copyWith(
        matchScope: 'matched_only',
        minScore: 80,
      );
      JobFilterState? updatedState;

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: JobFilterBar(
              filterState: modifiedState,
              onFilterChanged: (state) => updatedState = state,
              onOpenFilterDialog: () {},
            ),
          ),
        ),
      );

      expect(find.text('Filters (2)'), findsOneWidget);
      expect(find.text('Reset'), findsOneWidget);

      await tester.ensureVisible(find.text('Reset'));
      await tester.pumpAndSettle();

      await tester.tap(find.text('Reset'));
      await tester.pump();
      expect(updatedState, isNotNull);
      expect(updatedState!.isDefault, isTrue);
    });

    testWidgets('renders right scroll chevron when content overflows horizontally', (tester) async {
      const defaultState = JobFilterState();

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SizedBox(
              width: 320,
              child: JobFilterBar(
                filterState: defaultState,
                onFilterChanged: (_) {},
                onOpenFilterDialog: () {},
              ),
            ),
          ),
        ),
      );

      await tester.pumpAndSettle();

      expect(find.byIcon(Icons.chevron_right), findsOneWidget);
      await tester.tap(find.byIcon(Icons.chevron_right));
      await tester.pumpAndSettle();

      expect(find.byIcon(Icons.chevron_left), findsOneWidget);
    });
  });
}
