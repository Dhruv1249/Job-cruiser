
import 'package:dio/dio.dart';
import 'package:package_info_plus/package_info_plus.dart';

/// Represents a pending app update fetched from the GitHub Releases API.
class PendingUpdate {
  const PendingUpdate({
    required this.latestBuildNumber,
    required this.versionName,
    required this.releaseNotes,
    required this.downloadUrl,
    required this.tagName,
  });

  final int latestBuildNumber;
  final String versionName;
  final String releaseNotes;
  final String downloadUrl;
  final String tagName;
}

/// Polls the GitHub Releases API to detect whether a newer APK build is available.
///
/// Compares the installed [PackageInfo.buildNumber] (set by --build-number at compile time)
/// against the build number encoded in the latest release tag (format: `vX.Y.Z+<buildNumber>`).
/// Returns a [PendingUpdate] when a newer build exists, or null when the app is up to date.
class UpdateCheckerService {
  UpdateCheckerService({Dio? dio}) : _dio = dio ?? Dio();

  static const String _releasesApiUrl =
      'https://api.github.com/repos/Dhruv1249/Job-cruiser/releases/latest';

  static const String _apkAssetPattern = 'JobCruiser-';

  final Dio _dio;

  /// Fetches the latest GitHub release and returns a [PendingUpdate] if the
  /// installed build is older, or null if already up to date or on an error.
  Future<PendingUpdate?> checkForUpdate() async {
    try {
      final packageInfo = await PackageInfo.fromPlatform();
      final installedBuildNumber = int.tryParse(packageInfo.buildNumber) ?? 0;

      final response = await _dio.get<Map<String, dynamic>>(
        _releasesApiUrl,
        options: Options(
          headers: {'Accept': 'application/vnd.github+json'},
          receiveTimeout: const Duration(seconds: 10),
        ),
      );

      if (response.statusCode != 200 || response.data == null) {
        return null;
      }

      final releaseData = response.data!;
      final tagName = releaseData['tag_name'] as String? ?? '';
      final latestBuildNumber = _extractBuildNumber(tagName);

      if (latestBuildNumber <= installedBuildNumber) {
        return null;
      }

      final apkDownloadUrl = _extractApkDownloadUrl(releaseData);
      if (apkDownloadUrl.isEmpty) {
        return null;
      }

      final versionName = _extractVersionName(tagName);
      final releaseBody = releaseData['body'] as String? ?? '';
      final releaseNotes = _extractChangelog(releaseBody);

      return PendingUpdate(
        latestBuildNumber: latestBuildNumber,
        versionName: versionName,
        releaseNotes: releaseNotes,
        downloadUrl: apkDownloadUrl,
        tagName: tagName,
      );
    } catch (_) {
      return null;
    }
  }

  int _extractBuildNumber(String tagName) {
    final plusIndex = tagName.indexOf('+');
    if (plusIndex == -1) return 0;
    return int.tryParse(tagName.substring(plusIndex + 1)) ?? 0;
  }

  String _extractVersionName(String tagName) {
    final withoutV = tagName.startsWith('v') ? tagName.substring(1) : tagName;
    final plusIndex = withoutV.indexOf('+');
    return plusIndex == -1 ? withoutV : withoutV.substring(0, plusIndex);
  }

  String _extractApkDownloadUrl(Map<String, dynamic> releaseData) {
    final assets = releaseData['assets'] as List<dynamic>? ?? [];
    for (final asset in assets) {
      final assetMap = asset as Map<String, dynamic>;
      final name = assetMap['name'] as String? ?? '';
      if (name.startsWith(_apkAssetPattern) && name.endsWith('.apk')) {
        return assetMap['browser_download_url'] as String? ?? '';
      }
    }
    return '';
  }

  String _extractChangelog(String releaseBody) {
    const changelogMarker = '### What changed';
    final markerIndex = releaseBody.indexOf(changelogMarker);
    if (markerIndex == -1) return '';
    final afterMarker = releaseBody.substring(markerIndex + changelogMarker.length).trim();
    final nextSectionIndex = afterMarker.indexOf('\n###');
    return nextSectionIndex == -1
        ? afterMarker.trim()
        : afterMarker.substring(0, nextSectionIndex).trim();
  }
}
