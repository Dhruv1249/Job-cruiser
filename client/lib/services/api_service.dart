import 'package:flutter/foundation.dart';
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
  final String _baseUrl = _resolveBaseUrl();

  static String _resolveBaseUrl() {
    const compileTimeUrl = String.fromEnvironment('API_BASE_URL');
    if (compileTimeUrl.isNotEmpty) {
      return compileTimeUrl;
    }
    final envUrl = dotenv.env['API_BASE_URL'];
    if (envUrl != null && envUrl.isNotEmpty) {
      return envUrl;
    }
    if (kIsWeb) {
      final origin = Uri.base.origin;
      if (origin.isNotEmpty && !origin.startsWith('file://')) {
        return '$origin/api';
      }
    }
    return 'http://localhost:8080/api';
  }

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

  /// Fetches AI matched and raw jobs for the current user matching specified filter parameters.
  Future<List<MatchedJob>> fetchMatchedJobs({
    int minScore = 0,
    int maxScore = 100,
    int? days,
    String matchScope = 'all',
    bool remoteOnly = false,
    bool viewedOnly = false,
    bool unviewedOnly = false,
    String sortBy = 'score_desc',
    int limit = 50,
    int offset = 0,
  }) async {
    try {
      final queryParams = <String, dynamic>{
        'min_score': minScore,
        'max_score': maxScore,
        'match_scope': matchScope,
        'viewed_only': viewedOnly,
        'unviewed_only': unviewedOnly,
        'remote_only': remoteOnly,
        'sort_by': sortBy,
        'limit': limit,
        'offset': offset,
      };

      if (days != null && days > 0) {
        queryParams['days'] = days;
      }

      final response = await _dio.get(
        '/jobs/matched',
        queryParameters: queryParams,
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

  /// Removes an application from the tracking pipeline.
  Future<bool> deleteApplication(String applicationId) async {
    try {
      final response = await _dio.delete('/applications/$applicationId');
      return response.statusCode == 200;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return false;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Fetches a single job by its ID with full details and match metadata.
  Future<MatchedJob?> fetchJobById(String jobId) async {
    try {
      final response = await _dio.get('/jobs/$jobId');
      final data = response.data;
      if (data != null && data['data'] != null) {
        return MatchedJob.fromJson(data['data'] as Map<String, dynamic>);
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

  /// Dismisses / hides a job from the user's feed while keeping it in the database.
  Future<bool> dismissJob(String jobId) async {
    try {
      final response = await _dio.post('/jobs/$jobId/dismiss');
      return response.statusCode == 200;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return false;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Restores a previously dismissed job back to the user's feed.
  Future<bool> undismissJob(String jobId) async {
    try {
      final response = await _dio.post('/jobs/$jobId/undismiss');
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

  /// Fetches current AI match engine status (whether background processing is active).
  Future<Map<String, dynamic>> fetchMatchStatus() async {
    try {
      final response = await _dio.get('/jobs/match-status');
      if (response.data is Map<String, dynamic>) {
        return response.data as Map<String, dynamic>;
      }
      return {'is_evaluating': false, 'pending_count': 0};
    } catch (e) {
      _logger.e(e);
      return {'is_evaluating': false, 'pending_count': 0};
    }
  }

  /// Fetches active master keywords dictionary for Master Admin.
  Future<List<Map<String, dynamic>>> fetchMasterKeywordsForAdmin() async {
    try {
      final response = await _dio.get('/admin/keywords/master');
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

  /// Manually adds a new keyword to the master dictionary.
  Future<bool> addMasterKeyword(String keyword, String category) async {
    try {
      final response = await _dio.post('/admin/keywords/manual', data: {
        'keyword': keyword,
        'category': category,
      });
      return response.statusCode == 201 || response.statusCode == 200;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Deletes a master keyword by ID.
  Future<bool> deleteMasterKeyword(int id) async {
    try {
      final response = await _dio.delete('/admin/keywords/master/$id');
      return response.statusCode == 200;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Saves self-hosted open-overleaf configuration.
  Future<bool> saveOverleafConfig({
    required String deploymentUrl,
    String? mcpSecret,
    String? projectName,
    String? resumeTemplatePath,
    String? coverLetterTemplatePath,
  }) async {
    try {
      final payload = <String, dynamic>{
        'deployment_url': deploymentUrl,
      };
      if (mcpSecret != null && mcpSecret.isNotEmpty) {
        payload['mcp_secret'] = mcpSecret;
      }
      if (projectName != null && projectName.isNotEmpty) {
        payload['project_name'] = projectName;
      }
      if (resumeTemplatePath != null && resumeTemplatePath.isNotEmpty) {
        payload['resume_template_path'] = resumeTemplatePath;
      }
      if (coverLetterTemplatePath != null && coverLetterTemplatePath.isNotEmpty) {
        payload['cover_letter_template_path'] = coverLetterTemplatePath;
      }
      final response = await _dio.post('/overleaf/config', data: payload);
      return response.statusCode == 200;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Fetches available baseline LaTeX templates from Open-Overleaf.
  Future<Map<String, dynamic>?> fetchTemplates() async {
    try {
      final response = await _dio.get('/tailor/templates');
      if (response.data is Map<String, dynamic>) {
        return response.data as Map<String, dynamic>;
      }
      return null;
    } catch (e) {
      _logger.e(e);
      return null;
    }
  }

  /// Re-seeds default baseline resume and cover letter templates in Open-Overleaf.
  Future<bool> seedDefaultTemplates() async {
    try {
      final response = await _dio.post('/tailor/templates/seed');
      return response.statusCode == 200;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Fetches self-hosted open-overleaf configuration.
  Future<Map<String, dynamic>?> fetchOverleafConfig() async {
    try {
      final response = await _dio.get('/overleaf/config');
      if (response.data is Map<String, dynamic>) {
        return response.data as Map<String, dynamic>;
      }
      return null;
    } catch (e) {
      _logger.e(e);
      return null;
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

  /// Fetches scraper telemetry stats (Master Admin only).
  Future<Map<String, dynamic>?> fetchScraperStats() async {
    try {
      final response = await _dio.get('/admin/scraper-stats');
      if (response.data is Map<String, dynamic>) {
        return response.data as Map<String, dynamic>;
      }
      return null;
    } catch (e) {
      _logger.e(e);
      return null;
    }
  }

  /// Triggers AI resume tailoring for the given job and compiles via the user's open-overleaf instance.
  /// Returns the response map containing pdf_base64, version_id, pdf_web_url, and compile_result.
  Future<Map<String, dynamic>?> tailorResume({
    required String jobId,
    int targetPages = 1,
  }) async {
    try {
      final response = await _dio.post(
        '/tailor/resume',
        data: {
          'job_id': jobId,
          'target_pages': targetPages,
        },
        options: Options(receiveTimeout: const Duration(seconds: 120)),
      );
      if (response.data is Map<String, dynamic>) {
        return response.data as Map<String, dynamic>;
      }
      return null;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      if (e.response?.data is Map<String, dynamic>) {
        final data = Map<String, dynamic>.from(e.response!.data as Map);
        data['status_code'] = e.response?.statusCode;
        return data;
      }
      return {'error': e.message ?? 'Network error', 'status_code': e.response?.statusCode};
    } catch (e) {
      _logger.e(e);
      return {'error': e.toString()};
    }
  }

  /// Triggers AI cover letter generation for the given job and compiles via the user's open-overleaf instance.
  /// Returns the response map containing pdf_base64, cover_letter_id, pdf_web_url, and compile_result.
  Future<Map<String, dynamic>?> generateCoverLetter({required String jobId}) async {
    try {
      final response = await _dio.post(
        '/tailor/cover-letter',
        data: {'job_id': jobId},
        options: Options(receiveTimeout: const Duration(seconds: 120)),
      );
      if (response.data is Map<String, dynamic>) {
        return response.data as Map<String, dynamic>;
      }
      return null;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      if (e.response?.data is Map<String, dynamic>) {
        final data = Map<String, dynamic>.from(e.response!.data as Map);
        data['status_code'] = e.response?.statusCode;
        return data;
      }
      return {'error': e.message ?? 'Network error', 'status_code': e.response?.statusCode};
    } catch (e) {
      _logger.e(e);
      return {'error': e.toString()};
    }
  }

  /// Fetches the list of saved resume versions for the current user, ordered newest first.
  Future<List<Map<String, dynamic>>> fetchResumeVersions() async {
    try {
      final response = await _dio.get('/resume-versions');
      final data = response.data;
      if (data != null && data['data'] is List) {
        return List<Map<String, dynamic>>.from(data['data'] as List);
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

  /// Re-fetches the PDF base64 for a saved resume version by re-reading it from open-overleaf via MCP.
  Future<Map<String, dynamic>?> fetchResumeVersionPDF(String versionId) async {
    try {
      final response = await _dio.get(
        '/resume-versions/$versionId/pdf',
        options: Options(receiveTimeout: const Duration(seconds: 60)),
      );
      if (response.data is Map<String, dynamic>) {
        return response.data as Map<String, dynamic>;
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

  /// Deletes a resume version reference from Postgres. The LaTeX in open-overleaf is retained.
  Future<bool> deleteResumeVersion(String versionId) async {
    try {
      final response = await _dio.delete('/resume-versions/$versionId');
      return response.statusCode == 200;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return false;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Marks a resume version as the user's default, clearing the flag on all other versions.
  Future<bool> setDefaultResumeVersion(String versionId) async {
    try {
      final response = await _dio.put('/resume-versions/$versionId/default');
      return response.statusCode == 200;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return false;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Fetches the list of saved cover letter versions for the current user, ordered newest first.
  Future<List<Map<String, dynamic>>> fetchCoverLetterVersions() async {
    try {
      final response = await _dio.get('/cover-letters');
      final data = response.data;
      if (data != null && data['data'] is List) {
        return List<Map<String, dynamic>>.from(data['data'] as List);
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

  /// Re-fetches the PDF base64 for a saved cover letter version from open-overleaf.
  Future<Map<String, dynamic>?> fetchCoverLetterPDF(String coverLetterId) async {
    try {
      final response = await _dio.get(
        '/cover-letters/$coverLetterId/pdf',
        options: Options(receiveTimeout: const Duration(seconds: 60)),
      );
      if (response.data is Map<String, dynamic>) {
        return response.data as Map<String, dynamic>;
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

  /// Deletes a cover letter version reference from Postgres.
  Future<bool> deleteCoverLetterVersion(String coverLetterId) async {
    try {
      final response = await _dio.delete('/cover-letters/$coverLetterId');
      return response.statusCode == 200;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return false;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Triggers background asynchronous tailoring for both Resume and Cover Letter.
  /// Returns immediately with 202 Accepted acknowledgement.
  Future<Map<String, dynamic>?> tailorApplicationAsync({
    required String jobId,
    int? targetResumePages,
    int? targetCoverLetterPages,
  }) async {
    try {
      final payload = <String, dynamic>{'job_id': jobId};
      if (targetResumePages != null && targetResumePages > 0) {
        payload['target_resume_pages'] = targetResumePages;
      }
      if (targetCoverLetterPages != null && targetCoverLetterPages > 0) {
        payload['target_cover_letter_pages'] = targetCoverLetterPages;
      }
      final response = await _dio.post(
        '/tailor/application',
        data: payload,
      );
      if (response.data is Map<String, dynamic>) {
        return response.data as Map<String, dynamic>;
      }
      return null;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      if (e.response?.data is Map<String, dynamic>) {
        final data = Map<String, dynamic>.from(e.response!.data as Map);
        data['status_code'] = e.response?.statusCode;
        return data;
      }
      return {'error': e.message ?? 'Network error', 'status_code': e.response?.statusCode};
    } catch (e) {
      _logger.e(e);
      return {'error': e.toString()};
    }
  }

  /// Fetches user notifications.
  Future<List<Map<String, dynamic>>> fetchNotifications() async {
    try {
      final response = await _dio.get('/notifications');
      final data = response.data;
      if (data != null && data['data'] is List) {
        return List<Map<String, dynamic>>.from(data['data'] as List);
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

  /// Marks a specific notification as read.
  Future<bool> markNotificationRead(String id) async {
    try {
      final response = await _dio.post('/notifications/$id/read');
      return response.statusCode == 200;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return false;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Marks all user notifications as read.
  Future<bool> markAllNotificationsRead() async {
    try {
      final response = await _dio.post('/notifications/read-all');
      return response.statusCode == 200;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return false;
    } catch (e) {
      _logger.e(e);
      return false;
    }
  }

  /// Fetches count of unread notifications.
  Future<int> fetchUnreadNotificationsCount() async {
    try {
      final response = await _dio.get('/notifications/unread-count');
      final data = response.data;
      if (data != null && data['unread_count'] is int) {
        return data['unread_count'] as int;
      }
      return 0;
    } on DioException catch (e) {
      _logger.e(e.response?.data);
      return 0;
    } catch (e) {
      _logger.e(e);
      return 0;
    }
  }
}

