import 'package:flutter/material.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_app/profile.dart';
import 'package:flutter_app/preferences.dart';

void main() {
  setUpAll(() {
    dotenv.loadFromString(envString: 'API_BASE_URL=http://localhost:8080');
  });

  group('Version details rendering tests', () {
    testWidgets('ProfilePage renders app information section', (tester) async {
      tester.view.physicalSize = const Size(1200, 1800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      await tester.pumpWidget(
        const MaterialApp(
          home: ProfilePage(),
        ),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('APP INFORMATION'), findsOneWidget);
      expect(find.text('Application Version'), findsOneWidget);
      expect(find.text('Platform & Environment'), findsOneWidget);
    });

    testWidgets('SetPreferencesScreen renders app version section', (tester) async {
      tester.view.physicalSize = const Size(1200, 2400);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      await tester.pumpWidget(
        const MaterialApp(
          home: SetPreferencesScreen(),
        ),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('App Version & Environment'), findsOneWidget);
      expect(find.text('Platform'), findsOneWidget);
    });

    testWidgets('ProfilePage renders documents section matching full alignment width', (tester) async {
      tester.view.physicalSize = const Size(1200, 1800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      await tester.pumpWidget(
        const MaterialApp(
          home: ProfilePage(
            initialProfileData: {
              'full_name': 'Dhruv Dev',
              'primary_email': 'dhruv@example.com',
            },
            initialPreferencesData: {
              'target_roles': ['Backend Engineer'],
            },
          ),
        ),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      final matchingSetupFinder = find.ancestor(
        of: find.text('Matching Setup'),
        matching: find.byType(Container),
      ).first;
      final documentsContainerFinder = find.ancestor(
        of: find.byType(CircularProgressIndicator),
        matching: find.byType(Container),
      ).first;

      final matchingSetupWidth = tester.getSize(matchingSetupFinder).width;
      final documentsContainerWidth = tester.getSize(documentsContainerFinder).width;
      expect(documentsContainerWidth, equals(matchingSetupWidth));
    });

    testWidgets('SetPreferencesScreen renders USD and INR SegmentedButton when salary minimum is enabled', (tester) async {
      tester.view.physicalSize = const Size(1200, 3200);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      await tester.pumpWidget(
        const MaterialApp(
          home: SetPreferencesScreen(),
        ),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 200));

      final salarySwitchFinder = find.descendant(
        of: find.ancestor(
          of: find.text('ANY SALARY / NO MINIMUM'),
          matching: find.byType(Row),
        ),
        matching: find.byType(Switch),
      );
      expect(salarySwitchFinder, findsOneWidget);

      await tester.tap(salarySwitchFinder);
      await tester.pumpAndSettle();

      expect(find.byType(SegmentedButton<String>), findsOneWidget);
      expect(find.text('USD (\$)'), findsOneWidget);
      expect(find.text('INR (₹)'), findsOneWidget);
    });
  });
}
