import 'dart:convert';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('Admin Keywords Parsing Tests', () {
    test('splits comma-separated keywords with trimming and deduplication', () {
      const rawText = '  rust, Kubernetes, fastapi, rust, DOCKER ,  ';
      final tokens = rawText
          .split(RegExp(r'[,;\n]+'))
          .map((s) => s.trim())
          .where((s) => s.isNotEmpty)
          .toList();

      final seen = <String>{};
      final uniqueTokens = <String>[];
      for (final token in tokens) {
        final lower = token.toLowerCase();
        if (!seen.contains(lower)) {
          seen.add(lower);
          uniqueTokens.add(token);
        }
      }

      expect(uniqueTokens, equals(['rust', 'Kubernetes', 'fastapi', 'DOCKER']));
    });

    test('handles newline-separated or mixed comma/newline entries', () {
      const rawText = 'python\nflutter, golang;\naws';
      final tokens = rawText
          .split(RegExp(r'[,;\n]+'))
          .map((s) => s.trim())
          .where((s) => s.isNotEmpty)
          .toList();

      expect(tokens, equals(['python', 'flutter', 'golang', 'aws']));
    });

    test('handles empty or whitespace-only input safely', () {
      const rawText = '   ,  , \n  ;  ';
      final tokens = rawText
          .split(RegExp(r'[,;\n]+'))
          .map((s) => s.trim())
          .where((s) => s.isNotEmpty)
          .toList();

      expect(tokens, isEmpty);
    });
  });

  group('Admin Keywords Export Tests', () {
    final sampleMasterKeywords = [
      {'id': 1, 'keyword': 'fastapi', 'category': 'scraper', 'created_at': '2026-09-01'},
      {'id': 2, 'keyword': 'kubernetes', 'category': 'scraper', 'created_at': '2026-09-02'},
      {'id': 3, 'keyword': 'rust', 'category': 'scraper', 'created_at': '2026-09-03'},
    ];

    test('exports JSON keywords list correctly', () {
      final keywordStrings = sampleMasterKeywords
          .map((item) => (item['keyword'] as String? ?? '').trim())
          .where((k) => k.isNotEmpty)
          .toList();

      final jsonString = const JsonEncoder.withIndent('  ').convert(keywordStrings);
      final decoded = jsonDecode(jsonString) as List;

      expect(decoded, equals(['fastapi', 'kubernetes', 'rust']));
    });

    test('exports JSON full metadata correctly', () {
      final jsonString = const JsonEncoder.withIndent('  ').convert(sampleMasterKeywords);
      final decoded = jsonDecode(jsonString) as List;

      expect(decoded.length, equals(3));
      expect(decoded[0]['keyword'], equals('fastapi'));
      expect(decoded[0]['id'], equals(1));
    });

    test('exports TXT comma-separated correctly', () {
      final keywordStrings = sampleMasterKeywords
          .map((item) => (item['keyword'] as String? ?? '').trim())
          .where((k) => k.isNotEmpty)
          .toList();

      final txtContent = keywordStrings.join(', ');
      expect(txtContent, equals('fastapi, kubernetes, rust'));
    });

    test('exports TXT newline-separated correctly', () {
      final keywordStrings = sampleMasterKeywords
          .map((item) => (item['keyword'] as String? ?? '').trim())
          .where((k) => k.isNotEmpty)
          .toList();

      final txtContent = keywordStrings.join('\n');
      expect(txtContent, equals('fastapi\nkubernetes\nrust'));
    });
  });
}
