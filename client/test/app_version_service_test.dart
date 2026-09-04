import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_app/services/app_version_service.dart';

void main() {
  group('AppVersionDetails Model', () {
    test('formats displayVersion and compactVersion correctly', () {
      const details = AppVersionDetails(
        appName: 'Job Cruiser',
        version: '1.0.2',
        buildNumber: '14',
        platformName: 'Web',
      );

      expect(details.displayVersion, equals('v1.0.2 (Build 14) • Web'));
      expect(details.compactVersion, equals('v1.0.2+14'));
      expect(details.isWeb, isTrue);
    });

    test('handles fallback when fields are empty', () {
      const details = AppVersionDetails(
        appName: 'Job Cruiser',
        version: '',
        buildNumber: '',
        platformName: 'Android',
      );

      expect(details.displayVersion, equals('v1.0.0 • Android'));
      expect(details.compactVersion, equals('v1.0.0'));
      expect(details.isWeb, isFalse);
    });
  });

  group('AppVersionService Resolution', () {
    test('returns structured version details gracefully', () async {
      const service = AppVersionService();
      final details = await service.getVersionDetails();

      expect(details.appName.isNotEmpty, isTrue);
      expect(details.version.isNotEmpty, isTrue);
      expect(details.platformName.isNotEmpty, isTrue);
    });
  });
}
