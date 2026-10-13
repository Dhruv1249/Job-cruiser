import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_app/services/api_service.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    dotenv.loadFromString(envString: 'API_BASE_URL=http://localhost:8080/api');
    FlutterSecureStorage.setMockInitialValues({
      'jwt_token': 'sample-initial-jwt-token-12345',
    });
  });

  group('ApiService Session Token Management Tests', () {
    test('retrieves saved token successfully', () async {
      final apiService = ApiService();
      final retrievedToken = await apiService.getToken();
      expect(retrievedToken, equals('sample-initial-jwt-token-12345'));
    });

    test('saves renewed session token accurately', () async {
      final apiService = ApiService();
      const renewedToken = 'newly-refreshed-jwt-token-67890';
      await apiService.saveToken(renewedToken);

      final currentToken = await apiService.getToken();
      expect(currentToken, equals(renewedToken));
    });

    test('clears session token upon user logout', () async {
      final apiService = ApiService();
      await apiService.clearToken();

      final currentToken = await apiService.getToken();
      expect(currentToken, isNull);
    });
  });
}
