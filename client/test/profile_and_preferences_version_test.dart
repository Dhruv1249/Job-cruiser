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
  });
}
