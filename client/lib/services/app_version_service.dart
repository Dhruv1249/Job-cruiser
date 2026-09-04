import 'package:flutter/foundation.dart';
import 'package:package_info_plus/package_info_plus.dart';

/// Encapsulates application version details across web and mobile platforms.
class AppVersionDetails {
  const AppVersionDetails({
    required this.appName,
    required this.version,
    required this.buildNumber,
    required this.platformName,
  });

  final String appName;
  final String version;
  final String buildNumber;
  final String platformName;

  bool get isWeb => platformName.toLowerCase() == 'web';

  /// Returns a human-readable display string such as "v1.0.0 (Build 1) • Web".
  String get displayVersion {
    final versionPart = version.isNotEmpty ? 'v$version' : 'v1.0.0';
    final buildPart = buildNumber.isNotEmpty ? ' (Build $buildNumber)' : '';
    final platformPart = platformName.isNotEmpty ? ' • $platformName' : '';
    return '$versionPart$buildPart$platformPart';
  }

  /// Returns a concise version string such as "v1.0.0+1" or "v1.0.0".
  String get compactVersion {
    final versionPart = version.isNotEmpty ? 'v$version' : 'v1.0.0';
    final buildPart = buildNumber.isNotEmpty ? '+$buildNumber' : '';
    return '$versionPart$buildPart';
  }
}

/// Service resolving runtime package information with graceful fallbacks across web and mobile.
class AppVersionService {
  const AppVersionService();

  /// Fetches runtime version details from the active platform.
  Future<AppVersionDetails> getVersionDetails() async {
    try {
      final packageInfo = await PackageInfo.fromPlatform();
      final resolvedVersion = packageInfo.version.isNotEmpty ? packageInfo.version : '1.0.0';
      final resolvedBuild = packageInfo.buildNumber.isNotEmpty ? packageInfo.buildNumber : '1';
      final resolvedAppName = packageInfo.appName.isNotEmpty ? packageInfo.appName : 'Job Cruiser';
      final platformName = _resolvePlatformName();

      return AppVersionDetails(
        appName: resolvedAppName,
        version: resolvedVersion,
        buildNumber: resolvedBuild,
        platformName: platformName,
      );
    } catch (_) {
      return AppVersionDetails(
        appName: 'Job Cruiser',
        version: '1.0.0',
        buildNumber: '1',
        platformName: _resolvePlatformName(),
      );
    }
  }

  static String _resolvePlatformName() {
    if (kIsWeb) return 'Web';
    switch (defaultTargetPlatform) {
      case TargetPlatform.android:
        return 'Android';
      case TargetPlatform.iOS:
        return 'iOS';
      case TargetPlatform.macOS:
        return 'macOS';
      case TargetPlatform.windows:
        return 'Windows';
      case TargetPlatform.linux:
        return 'Linux';
      default:
        return 'Client';
    }
  }
}
