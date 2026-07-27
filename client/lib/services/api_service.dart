import 'package:dio/dio.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:logger/logger.dart';
import '../models/job.dart';
import '../models/application.dart';

/// Central API Service handling network interactions with the Go Backend.
class ApiService {
  final Dio _dio = Dio();
  final FlutterSecureStorage _storage = const FlutterSecureStorage();
  final Logger _logger = Logger();

  static const String _tokenKey = 'jwt_token';
  final String _baseUrl = dotenv.env['API_BASE_URL'] ?? "http://192.168.1.12:8080/api";

  ApiService() {
    _dio.options.baseUrl = _baseUrl;
    _dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
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

  /// Clears stored JWT token upon user logout.
  Future<void> clearToken() async {
    await _storage.delete(key: _tokenKey);
  }

  /// Persists authentication token.
  Future<void> saveToken(String token) async {
    await _storage.write(key: _tokenKey, value: token);
  }

  /// Retrieves saved authentication token.
  Future<String?> getToken() async {
    return await _storage.read(key: _tokenKey);
  }

  /// Fetches authenticated user profile.
  Future<Map<String, dynamic>?> fetchProfile() async {
    try {
      final response = await _dio.get('/user/me');
      return response.data as Map<String, dynamic>?;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return null;
    } catch (e) {
      _logger.e(e);
      return null;
    }
  }

  /// Authenticates user with email and password.
  Future<Map<String, dynamic>?> login(String email, String password) async {
    try {
      final response = await _dio.post('/login', data: {
        'primary_email': email,
        'password': password,
      });
      final data = response.data;
      if (data != null && data['token'] != null) {
        await saveToken(data['token'] as String);
        return data as Map<String, dynamic>;
      }
      return null;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return null;
    } catch (e) {
      _logger.e(e);
      return null;
    }
  }

  /// Registers a new user account.
  Future<bool> signup(String name, String email, String password) async {
    try {
      final response = await _dio.post('/signup', data: {
        'full_name': name,
        'primary_email': email,
        'password': password,
      });
      final data = response.data;
      if (data['token'] != null) {
        await saveToken(data['token'] as String);
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

  /// Authenticates user via Google SSO.
  Future<Map<String, dynamic>?> googleLogin(String idToken) async {
    try {
      final response = await _dio.post('/auth/google', data: {
        'id_token': idToken,
      });
      final data = response.data;
      if (data != null && data['token'] != null) {
        await saveToken(data['token'] as String);
        return data as Map<String, dynamic>;
      }
      return null;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return null;
    } catch (e) {
      _logger.e(e);
      return null;
    }
  }

  /// Fetches raw scraped jobs from the backend.
  Future<List<MatchedJob>> fetchRawJobs({int limit = 50, int page = 1}) async {
    try {
      final response = await _dio.get(
        '/jobs',
        queryParameters: {'limit': limit, 'page': page},
      );
      final data = response.data;
      if (data != null && data['data'] is List) {
        final List jobsList = data['data'] as List;
        return jobsList
            .map((item) => MatchedJob.fromJson(item as Map<String, dynamic>))
            .toList();
      }
      return [];
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return [];
    } catch (e) {
      _logger.e(e);
      return [];
    }
  }

  /// Fetches AI matched jobs for the current user.
  Future<List<MatchedJob>> fetchMatchedJobs({
    int minScore = 0,
    bool viewedOnly = false,
    bool unviewedOnly = false,
    int limit = 50,
    int offset = 0,
  }) async {
    try {
      final response = await _dio.get(
        '/jobs/matched',
        queryParameters: {
          'min_score': minScore,
          'viewed_only': viewedOnly,
          'unviewed_only': unviewedOnly,
          'limit': limit,
          'offset': offset,
        },
      );

      final data = response.data;
      if (data != null && data['jobs'] is List) {
        final List jobsList = data['jobs'] as List;
        return jobsList
            .map((item) => MatchedJob.fromJson(item as Map<String, dynamic>))
            .toList();
      }
      return [];
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return [];
    } catch (e) {
      _logger.e(e);
      return [];
    }
  }

  /// Marks a specific job as viewed by the authenticated user.
  Future<bool> markJobAsViewed(String jobId) async {
    if (jobId.isEmpty) return false;
    try {
      final response = await _dio.post('/jobs/$jobId/view');
      return response.statusCode == 200;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return false;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Fetches user applications tracked in the CRM.
  Future<List<JobApplication>> fetchApplications() async {
    try {
      final response = await _dio.get('/applications');
      final data = response.data;
      if (data != null && data['data'] is List) {
        final List appsList = data['data'] as List;
        return appsList
            .map((item) => JobApplication.fromJson(item as Map<String, dynamic>))
            .toList();
      }
      return [];
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return [];
    } catch (e) {
      _logger.e(e);
      return [];
    }
  }

  /// Creates a new job application entry in the pipeline.
  Future<bool> createApplication(String jobId, String status) async {
    try {
      final response = await _dio.post('/applications', data: {
        'job_id': jobId,
        'status': status,
      });
      return response.statusCode == 201 || response.statusCode == 200;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return false;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Updates the status of an existing application.
  Future<bool> updateApplicationStatus(String applicationId, String status) async {
    try {
      final response = await _dio.put(
        '/applications/$applicationId/status',
        data: {'status': status},
      );
      return response.statusCode == 200;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return false;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Fetches saved user preferences.
  Future<Map<String, dynamic>?> fetchPreferences() async {
    try {
      final response = await _dio.get('/preferences');
      final data = response.data;
      if (data != null && data['data'] != null) {
        final Map<String, dynamic> res = Map<String, dynamic>.from(data['data'] as Map);
        res['has_preferences'] = data['has_preferences'] ?? false;
        return res;
      }
      return null;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return null;
    } catch (e) {
      _logger.e(e);
      return null;
    }
  }

  /// Saves user preferences to the backend.
  Future<bool> savePreferences(Map<String, dynamic> preferenceData) async {
    try {
      final response = await _dio.post('/preferences', data: preferenceData);
      return response.statusCode == 200;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return false;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Parses raw CV text using Gemini AI via backend endpoint.
  Future<Map<String, dynamic>?> parseCVWithGemini(String rawCVText) async {
    try {
      final response = await _dio.post('/user/parse-cv', data: {
        'raw_cv_text': rawCVText,
      });
      final data = response.data;
      if (data != null && data['data'] != null) {
        return Map<String, dynamic>.from(data['data'] as Map);
      }
      return null;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return null;
    } catch (e) {
      _logger.e(e);
      return null;
    }
  }

  /// Fetches whitelisted email addresses for Master Admin.
  Future<List<Map<String, dynamic>>> fetchWhitelistedEmails() async {
    try {
      final response = await _dio.get('/admin/whitelisted-emails');
      final data = response.data;
      if (data != null && data['data'] is List) {
        return List<Map<String, dynamic>>.from(data['data'] as List);
      }
      return [];
    } catch (e) {
      _logger.e(e);
      return [];
    }
  }

  /// Adds a new email to the access whitelist.
  Future<bool> addWhitelistedEmail(String email, String notes) async {
    try {
      final response = await _dio.post('/admin/whitelisted-emails', data: {
        'email': email,
        'notes': notes,
      });
      return response.statusCode == 201 || response.statusCode == 200;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Removes an email from the whitelist.
  Future<bool> deleteWhitelistedEmail(String id) async {
    try {
      final response = await _dio.delete('/admin/whitelisted-emails/$id');
      return response.statusCode == 200;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Fetches pending keyword recommendations.
  Future<List<Map<String, dynamic>>> fetchPendingKeywords() async {
    try {
      final response = await _dio.get('/admin/keywords/pending');
      final data = response.data;
      if (data != null && data['data'] is List) {
        return List<Map<String, dynamic>>.from(data['data'] as List);
      }
      return [];
    } catch (e) {
      _logger.e(e);
      return [];
    }
  }

  /// Approves or rejects a pending keyword suggestion.
  Future<bool> approveKeyword(String suggestionId, bool approve) async {
    try {
      final response = await _dio.post('/admin/keywords/approve', data: {
        'suggestion_id': suggestionId,
        'approve': approve,
      });
      return response.statusCode == 200;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Saves self-hosted open-overleaf configuration.
  Future<bool> saveOverleafConfig({
    required String deploymentUrl,
    required String githubUsername,
    required String githubRepoName,
    required String accessToken,
  }) async {
    try {
      final response = await _dio.post('/overleaf/config', data: {
        'deployment_url': deploymentUrl,
        'github_username': githubUsername,
        'github_repo_name': githubRepoName,
        'access_token': accessToken,
      });
      return response.statusCode == 200;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Fetches registered users for Master Admin management.
  Future<List<Map<String, dynamic>>> fetchUsersForAdmin() async {
    try {
      final response = await _dio.get('/admin/users');
      final data = response.data;
      if (data != null && data['data'] is List) {
        return List<Map<String, dynamic>>.from(data['data'] as List);
      }
      return [];
    } catch (e) {
      _logger.e(e);
      return [];
    }
  }

  /// Toggles AI matching enabled setting for a specific user (Master Admin only).
  Future<bool> toggleUserAIMatching(String userId, bool enabled) async {
    try {
      final response = await _dio.put('/admin/users/$userId/ai-matching', data: {
        'enabled': enabled,
      });
      return response.statusCode == 200;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }
}
