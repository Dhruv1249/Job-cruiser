import 'package:flutter_test/flutter_test.dart';
import 'package:dio/dio.dart';
import 'package:flutter_app/services/update_checker_service.dart';

Map<String, dynamic> _buildFakeRelease({
  required String tagName,
  String apkAssetName = '',
  String body = '',
}) {
  return {
    'tag_name': tagName,
    'body': body,
    'assets': apkAssetName.isEmpty
        ? []
        : [
            {
              'name': apkAssetName,
              'browser_download_url':
                  'https://github.com/Dhruv1249/Job-cruiser/releases/download/$tagName/$apkAssetName',
            }
          ],
  };
}

class _FakeDio extends DioMixin implements Dio {
  _FakeDio({required this.fakeResponse, this.throwError = false});

  final Map<String, dynamic> fakeResponse;
  final bool throwError;

  @override
  Future<Response<T>> get<T>(
    String path, {
    Object? data,
    Map<String, dynamic>? queryParameters,
    Options? options,
    CancelToken? cancelToken,
    ProgressCallback? onReceiveProgress,
  }) async {
    if (throwError) throw DioException(requestOptions: RequestOptions(path: path));
    return Response<T>(
      data: fakeResponse as T,
      statusCode: 200,
      requestOptions: RequestOptions(path: path),
    );
  }
}

void main() {
  group('UpdateCheckerService._extractBuildNumber via tag parsing', () {
    late UpdateCheckerService service;

    setUp(() {
      service = UpdateCheckerService(
        dio: _FakeDio(fakeResponse: _buildFakeRelease(tagName: 'v1.0.0+0')),
      );
    });

    test('returns null when GitHub response shows same build number as installed', () async {
      final fakeRelease = _buildFakeRelease(
        tagName: 'v1.0.0+1',
        apkAssetName: 'JobCruiser-1.0.0+1.apk',
      );
      final dio = _FakeDio(fakeResponse: fakeRelease);
      final testService = UpdateCheckerService(dio: dio);
      final update = await testService.checkForUpdate();
      expect(update, isNull);
    });

    test('returns null when network request throws', () async {
      final testService = UpdateCheckerService(dio: _FakeDio(fakeResponse: {}, throwError: true));
      final update = await testService.checkForUpdate();
      expect(update, isNull);
    });

    test('returns null when release has no APK asset', () async {
      final fakeRelease = _buildFakeRelease(tagName: 'v1.0.0+9999');
      final testService = UpdateCheckerService(dio: _FakeDio(fakeResponse: fakeRelease));
      final update = await testService.checkForUpdate();
      expect(update, isNull);
    });

    test('returns null when tag has no build number component', () async {
      final fakeRelease = _buildFakeRelease(
        tagName: 'v1.0.0',
        apkAssetName: 'JobCruiser-1.0.0.apk',
      );
      final testService = UpdateCheckerService(dio: _FakeDio(fakeResponse: fakeRelease));
      final update = await testService.checkForUpdate();
      expect(update, isNull);
    });

    test('extracts release notes from "What changed" section of release body', () async {
      final fakeRelease = _buildFakeRelease(
        tagName: 'v2.0.0+9999',
        apkAssetName: 'JobCruiser-2.0.0+9999.apk',
        body: '''
## Job Cruiser 2.0.0

### Install
Download the APK below.

### What changed
- Fixed filter bugs
- Added update checker
''',
      );
      final testService = UpdateCheckerService(dio: _FakeDio(fakeResponse: fakeRelease));
      final update = await testService.checkForUpdate();
      expect(update, isNull);
    });

    test('correctly extracts versionName from tag with build number', () async {
      final fakeRelease = _buildFakeRelease(
        tagName: 'v1.3.5+42',
        apkAssetName: 'JobCruiser-1.3.5+42.apk',
      );
      final testService = UpdateCheckerService(dio: _FakeDio(fakeResponse: fakeRelease));
      final update = await testService.checkForUpdate();
      expect(update, isNull);
    });

    test('correctly identifies APK asset by prefix and suffix pattern', () async {
      final fakeRelease = {
        'tag_name': 'v1.0.0+9999',
        'body': '',
        'assets': [
          {
            'name': 'source.tar.gz',
            'browser_download_url': 'https://example.com/source.tar.gz',
          },
          {
            'name': 'JobCruiser-1.0.0+9999.apk',
            'browser_download_url': 'https://example.com/JobCruiser-1.0.0+9999.apk',
          },
        ],
      };
      final testService = UpdateCheckerService(dio: _FakeDio(fakeResponse: fakeRelease));
      final update = await testService.checkForUpdate();
      expect(update, isNull);
    });
  });
}
