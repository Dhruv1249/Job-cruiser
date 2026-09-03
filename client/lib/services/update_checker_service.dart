import 'dart:io';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:open_file/open_file.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:path_provider/path_provider.dart';
import 'package:url_launcher/url_launcher.dart';

/// Represents a pending app update fetched from the GitHub Releases API.
class PendingUpdate {
  const PendingUpdate({
    required this.versionName,
    required this.releaseNotes,
    required this.downloadUrl,
    required this.tagName,
    this.latestBuildNumber = 0,
  });

  final String versionName;
  final String releaseNotes;
  final String downloadUrl;
  final String tagName;
  final int latestBuildNumber;
}

/// Polls the GitHub Releases API to detect whether a newer version is available.
///
/// Compares semantic versions (e.g., 1.0.0, 1.0.12, 1.1.0, 1.9.12) and supports
/// in-app direct APK download and package installer triggering.
class UpdateCheckerService {
  UpdateCheckerService({Dio? dio}) : _dio = dio ?? Dio();

  static const String _releasesApiUrl =
      'https://api.github.com/repos/Dhruv1249/Job-cruiser/releases/latest';

  static const String _apkAssetPattern = 'JobCruiser-';

  final Dio _dio;

  /// Fetches the latest GitHub release and returns a [PendingUpdate] if the
  /// installed version is older, or null if already up to date or on an error.
  Future<PendingUpdate?> checkForUpdate() async {
    if (kIsWeb) return null;
    try {
      final packageInfo = await PackageInfo.fromPlatform();
      final installedVersion = packageInfo.version;
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
      final latestVersionName = _extractVersionName(tagName);
      final latestBuildNumber = _extractBuildNumber(tagName);

      final hasNewerVersion = isVersionNewer(latestVersionName, installedVersion) ||
          (isVersionEqual(latestVersionName, installedVersion) &&
              latestBuildNumber > installedBuildNumber);

      if (!hasNewerVersion) {
        return null;
      }

      final apkDownloadUrl = _extractApkDownloadUrl(releaseData);
      if (apkDownloadUrl.isEmpty) {
        return null;
      }

      final releaseBody = releaseData['body'] as String? ?? '';
      final releaseNotes = _extractChangelog(releaseBody);

      return PendingUpdate(
        versionName: latestVersionName,
        releaseNotes: releaseNotes,
        downloadUrl: apkDownloadUrl,
        tagName: tagName,
        latestBuildNumber: latestBuildNumber,
      );
    } catch (_) {
      return null;
    }
  }

  /// Downloads the APK binary directly and triggers the system package installer.
  Future<bool> downloadAndInstall({
    required PendingUpdate update,
    void Function(double progress)? onProgress,
  }) async {
    if (kIsWeb) {
      final uri = Uri.tryParse(update.downloadUrl);
      if (uri == null) return false;
      return await launchUrl(uri, mode: LaunchMode.externalApplication);
    }

    try {
      final tempDirectory = await getTemporaryDirectory();
      final targetFilePath =
          '${tempDirectory.path}/JobCruiser-v${update.versionName}.apk';

      final file = File(targetFilePath);
      if (await file.exists()) {
        await file.delete();
      }

      await _dio.download(
        update.downloadUrl,
        targetFilePath,
        onReceiveProgress: (receivedBytes, totalBytes) {
          if (totalBytes > 0 && onProgress != null) {
            onProgress(receivedBytes / totalBytes);
          }
        },
      );

      final openResult = await OpenFile.open(
        targetFilePath,
        type: 'application/vnd.android.package-archive',
      );

      return openResult.type == ResultType.done;
    } catch (_) {
      return false;
    }
  }

  /// Returns true if [latestVersionString] is strictly newer than [currentVersionString].
  static bool isVersionNewer(String latestVersionString, String currentVersionString) {
    final latestParts = _parseSemanticVersion(latestVersionString);
    final currentParts = _parseSemanticVersion(currentVersionString);

    for (var index = 0; index < 3; index++) {
      if (latestParts[index] > currentParts[index]) return true;
      if (latestParts[index] < currentParts[index]) return false;
    }

    return false;
  }

  /// Returns true if both semantic versions are equal.
  static bool isVersionEqual(String firstVersionString, String secondVersionString) {
    final firstParts = _parseSemanticVersion(firstVersionString);
    final secondParts = _parseSemanticVersion(secondVersionString);

    return firstParts[0] == secondParts[0] &&
        firstParts[1] == secondParts[1] &&
        firstParts[2] == secondParts[2];
  }

  static List<int> _parseSemanticVersion(String versionString) {
    var cleanedVersion = versionString.trim();
    if (cleanedVersion.startsWith('v') || cleanedVersion.startsWith('V')) {
      cleanedVersion = cleanedVersion.substring(1);
    }
    final plusIndex = cleanedVersion.indexOf('+');
    if (plusIndex != -1) {
      cleanedVersion = cleanedVersion.substring(0, plusIndex);
    }
    final segments = cleanedVersion.split('.');
    final major = segments.isNotEmpty ? (int.tryParse(segments[0]) ?? 0) : 0;
    final minor = segments.length > 1 ? (int.tryParse(segments[1]) ?? 0) : 0;
    final patch = segments.length > 2 ? (int.tryParse(segments[2]) ?? 0) : 0;
    return [major, minor, patch];
  }

  static String _extractVersionName(String tagName) {
    var withoutV = tagName.trim();
    if (withoutV.startsWith('v') || withoutV.startsWith('V')) {
      withoutV = withoutV.substring(1);
    }
    final plusIndex = withoutV.indexOf('+');
    return plusIndex == -1 ? withoutV : withoutV.substring(0, plusIndex);
  }

  static int _extractBuildNumber(String tagName) {
    final plusIndex = tagName.indexOf('+');
    if (plusIndex == -1) return 0;
    return int.tryParse(tagName.substring(plusIndex + 1)) ?? 0;
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
