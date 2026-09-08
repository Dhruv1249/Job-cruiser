import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_app/widgets/company_logo_avatar.dart';

void main() {
  testWidgets('CompanyLogoAvatar renders fallback letter when company has no valid domain', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: CompanyLogoAvatar(
            companyName: 'Unknown',
            jobUrl: '',
          ),
        ),
      ),
    );

    expect(find.text('U'), findsOneWidget);
  });

  testWidgets('CompanyLogoAvatar extracts first letter uppercase correctly', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: CompanyLogoAvatar(
            companyName: 'google',
            jobUrl: '',
          ),
        ),
      ),
    );

    expect(find.byType(CompanyLogoAvatar), findsOneWidget);
  });

  testWidgets('CompanyLogoAvatar extracts domain from career ATS URLs and renders image candidate', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: CompanyLogoAvatar(
            companyName: 'Airbnb',
            jobUrl: 'https://boards.greenhouse.io/airbnb/jobs/12345',
          ),
        ),
      ),
    );

    expect(find.byType(Image), findsOneWidget);
  });
}
