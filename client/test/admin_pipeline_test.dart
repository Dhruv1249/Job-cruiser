import 'package:flutter_test/flutter_test.dart';

void main() {
  group('Admin AI Pipeline Status Tests', () {
    test('parses active status correctly', () {
      final payload = {
        'is_permanently_stopped': false,
        'status': 'active',
      };

      final isStopped = payload['is_permanently_stopped'] == true;
      final status = payload['status'] as String?;

      expect(isStopped, isFalse);
      expect(status, equals('active'));
    });

    test('parses stopped status correctly', () {
      final payload = {
        'is_permanently_stopped': true,
        'status': 'stopped',
      };

      final isStopped = payload['is_permanently_stopped'] == true;
      final status = payload['status'] as String?;

      expect(isStopped, isTrue);
      expect(status, equals('stopped'));
    });

    test('defaults to active when status payload is missing or empty', () {
      final Map<String, dynamic> emptyPayload = {};

      final isStopped = emptyPayload['is_permanently_stopped'] == true;

      expect(isStopped, isFalse);
    });
  });
}
