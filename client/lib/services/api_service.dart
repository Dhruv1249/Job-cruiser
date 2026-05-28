import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:logger/logger.dart';

class ApiService {
  // Private variables start with an underscore in Dart
  final Dio _dio = Dio();
  final FlutterSecureStorage _storage = const FlutterSecureStorage();
  final Logger _logger = Logger();

  // The key we will use to store the JWT in the secure enclave
  static const String _tokenKey = 'jwt_token';

  final String _baseUrl = 'http://localhost:8080/api';

  ApiService() {
    // Configure the base URL
    _dio.options.baseUrl = _baseUrl;

    // Interceptor (middleware)
    _dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
          // Add the JWT token to the request headers
          final token = await getToken();
          if (token != null) {
            options.headers['Authorization'] = 'Bearer $token';
          }
          return handler.next(options);
        },
        onError: (error, handler) async {
          return handler.next(error);
        },
      ),
    );

    _dio.interceptors.add(
      LogInterceptor(
        requestBody: true,
        responseBody: true,
        requestHeader: true,
        responseHeader: true,
        request: true,
        error: true,
      ),
    );
  }

  // --- STORAGE METHODS ---
  Future<void> clearToken() async {
    await _storage.delete(key: _tokenKey);
  }

  Future<void> saveToken(String token) async {
    await _storage.write(key: _tokenKey, value: token);
  }

  Future<String?> getToken() async {
    return await _storage.read(key: _tokenKey);
  }

  // --- API METHODS ---

  Future<Map<String, dynamic>?> fetchProfile() async {
    try {
      final response = await _dio.get('/user/me');
      return response.data;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return null;
    } catch (e) {
      _logger.e(e);
      return null;
    }
  }

  Future<bool> login(String email, String password) async {
    try {
      final response = await _dio.post('/login', data: {
        'primary_email': email,
        'password': password,
      });
      final data = response.data;
      if (data['token'] != null) {
        await saveToken(data['token']);
        return true;
      }
      return false;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return false;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  Future<bool> signup(String name, String email, String password) async {
    try {
      final response = await _dio.post('/signup', data: {
        'full_name': name,
        'primary_email': email,
        'password': password,
      });
      final data = response.data;
      if (data['token'] != null) {
        await saveToken(data['token']);
        return true;
      }
      return false;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return false;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }


  Future<bool> googleLogin(String idToken) async {
    try {
      final response = await _dio.post('/auth/google', data: {
        'id_token': idToken,
      });
      final data = response.data;
      if (data['token'] != null) {
        await saveToken(data['token']);
        return true;
      }
      return false;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return false;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }
}
