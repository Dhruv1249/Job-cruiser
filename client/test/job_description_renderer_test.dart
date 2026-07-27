import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_app/widgets/job_description_renderer.dart';

void main() {
  test('unescapeHtmlEntities converts &nbsp; and HTML entities properly', () {
    const raw = '&nbsp;We are looking for an&nbsp;Analyst &amp; Engineer&nbsp;';
    final unescaped = unescapeHtmlEntities(raw);
    expect(unescaped, equals(' We are looking for an Analyst & Engineer '));
  });

  test('detectJobDescFormat detects HTML tags correctly', () {
    const htmlSnippet = '<div><h3>Backend Engineer</h3><p>We are hiring!</p></div>';
    expect(detectJobDescFormat(htmlSnippet), equals(JobDescFormat.html));
  });

  test('detectJobDescFormat detects Markdown syntax correctly', () {
    const markdownSnippet = '# Senior Developer\n\n- Experience with Go\n- Experience with Flutter\n\n[Apply here](https://example.com)';
    expect(detectJobDescFormat(markdownSnippet), equals(JobDescFormat.markdown));
  });

  test('detectJobDescFormat defaults to plain text for plain descriptions', () {
    const plainSnippet = 'Senior Full Stack Engineer position in New York.\nSalary: \$160k-\$210k + equity.';
    expect(detectJobDescFormat(plainSnippet), equals(JobDescFormat.plainText));
  });

  testWidgets('JobDescriptionRenderer renders format tag and content', (WidgetTester tester) async {
    const content = '# Title\n\n- Requirement 1';
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: JobDescriptionRenderer(content: content),
        ),
      ),
    );

    expect(find.text('Job Description'), findsOneWidget);
    expect(find.text('MARKDOWN'), findsOneWidget);
  });
}
