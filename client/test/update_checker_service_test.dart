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
  group('Semantic version comparison logic', () {
    test('detects patch version increment', () {
      expect(UpdateCheckerService.isVersionNewer('1.0.1', '1.0.0'), isTrue);
      expect(UpdateCheckerService.isVersionNewer('1.0.12', '1.0.1'), isTrue);
      expect(UpdateCheckerService.isVersionNewer('1.0.12', '1.0.11'), isTrue);
      expect(UpdateCheckerService.isVersionNewer('1.0.1', '1.0.12'), isFalse);
    });

    test('detects minor version increment', () {
      expect(UpdateCheckerService.isVersionNewer('1.1.0', '1.0.12'), isTrue);
      expect(UpdateCheckerService.isVersionNewer('1.9.12', '1.1.0'), isTrue);
      expect(UpdateCheckerService.isVersionNewer('1.1.0', '1.2.0'), isFalse);
    });

    test('detects major version increment', () {
      expect(UpdateCheckerService.isVersionNewer('2.0.0', '1.9.12'), isTrue);
      expect(UpdateCheckerService.isVersionNewer('1.9.12', '2.0.0'), isFalse);
    });

    test('handles leading v prefix and build numbers', () {
      expect(UpdateCheckerService.isVersionNewer('v1.0.12', '1.0.0'), isTrue);
      expect(UpdateCheckerService.isVersionNewer('v1.1.0+5', '1.0.12+1'), isTrue);
      expect(UpdateCheckerService.isVersionNewer('1.0.0', 'v1.0.0'), isFalse);
    });

    test('isVersionEqual identifies identical versions', () {
      expect(UpdateCheckerService.isVersionEqual('1.0.0', '1.0.0'), isTrue);
      expect(UpdateCheckerService.isVersionEqual('v1.0.12', '1.0.12'), isTrue);
      expect(UpdateCheckerService.isVersionEqual('v1.0.12+4', '1.0.12+9'), isTrue);
      expect(UpdateCheckerService.isVersionEqual('1.0.0', '1.0.1'), isFalse);
    });
  });

  group('UpdateCheckerService API responses', () {
    test('returns null when network request throws', () async {
      final testService = UpdateCheckerService(dio: _FakeDio(fakeResponse: {}, throwError: true));
      final update = await testService.checkForUpdate();
      expect(update, isNull);
    });

    test('returns null when release has no APK asset', () async {
      final fakeRelease = _buildFakeRelease(tagName: 'v9.9.9');
      final testService = UpdateCheckerService(dio: _FakeDio(fakeResponse: fakeRelease));
      final update = await testService.checkForUpdate();
      expect(update, isNull);
    });
  });
}
