import 'dart:async';
import 'dart:convert';

import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';

import 'api_service.dart';

const String _notificationChannelId = 'job_cruiser_high_match';
const String _notificationChannelName = 'High Match Alerts';
const String _notificationChannelDescription =
    'Push alerts when a job matches above your threshold score.';

/// Handles FCM token registration, notification permission, and routing of
/// incoming messages to the correct destination in the app.
///
/// Call [initialize] once from main() after Firebase.initializeApp().
/// Call [registerTokenWithBackend] after every successful login so the
/// backend always has the current token for this device.
class FCMService {
  FCMService._();

  static final FCMService instance = FCMService._();

  final FirebaseMessaging _messaging = FirebaseMessaging.instance;

  static const AndroidNotificationChannel _highImportanceChannel =
      AndroidNotificationChannel(
    _notificationChannelId,
    _notificationChannelName,
    description: _notificationChannelDescription,
    importance: Importance.max,
  );

  final FlutterLocalNotificationsPlugin _localNotifications =
      FlutterLocalNotificationsPlugin();

  /// Callback invoked when the user taps a notification while the app is
  /// running. The [jobId] is extracted from the FCM data payload.
  void Function(String jobId)? onNotificationTap;

  /// Requests notification permission from the OS, initialises the local
  /// notification channel, subscribes to foreground and tap events, and
  /// attempts to fetch the current FCM registration token.
  Future<void> initialize() async {
    if (kIsWeb) return;

    await _messaging.requestPermission(alert: true, badge: true, sound: true);

    await _localNotifications
        .resolvePlatformSpecificImplementation<
            AndroidFlutterLocalNotificationsPlugin>()
        ?.createNotificationChannel(_highImportanceChannel);

    const AndroidInitializationSettings androidSettings =
        AndroidInitializationSettings('@mipmap/ic_launcher');
    await _localNotifications.initialize(
      settings: const InitializationSettings(android: androidSettings),
      onDidReceiveNotificationResponse: _onLocalNotificationTap,
    );

    FirebaseMessaging.onMessage.listen(_handleForegroundMessage);
    FirebaseMessaging.onMessageOpenedApp.listen(_handleMessageTap);

    final RemoteMessage? initialMessage =
        await _messaging.getInitialMessage();
    if (initialMessage != null) {
      _handleMessageTap(initialMessage);
    }

    _messaging.onTokenRefresh.listen(_onTokenRefresh);
  }

  /// Retrieves the current FCM device token and POSTs it to the backend.
  /// Safe to call every login — the backend upserts on the user record.
  Future<void> registerTokenWithBackend(ApiService apiService) async {
    if (kIsWeb) return;

    try {
      final String? token = await _messaging.getToken();
      if (token == null || token.isEmpty) return;
      await apiService.putFCMToken(token);
    } catch (_) {}
  }

  void _handleForegroundMessage(RemoteMessage message) {
    final String? title = message.data['title'] as String?;
    final String? body = message.data['body'] as String?;
    if (title == null && body == null) return;

    unawaited(_localNotifications.show(
      id: message.hashCode,
      title: title,
      body: body,
      notificationDetails: NotificationDetails(
        android: AndroidNotificationDetails(
          _notificationChannelId,
          _notificationChannelName,
          channelDescription: _notificationChannelDescription,
          importance: Importance.max,
          priority: Priority.max,
          playSound: true,
          enableVibration: true,
          visibility: NotificationVisibility.public,
        ),
      ),
      payload: jsonEncode(message.data),
    ));
  }

  void _handleMessageTap(RemoteMessage message) {
    final String? jobId = message.data['job_id'] as String?;
    if (jobId != null && jobId.isNotEmpty && onNotificationTap != null) {
      onNotificationTap!(jobId);
    }
  }

  void _onLocalNotificationTap(NotificationResponse response) {
    if (response.payload == null) return;
    try {
      final Map<String, dynamic> data =
          jsonDecode(response.payload!) as Map<String, dynamic>;
      final String? jobId = data['job_id'] as String?;
      if (jobId != null && jobId.isNotEmpty && onNotificationTap != null) {
        onNotificationTap!(jobId);
      }
    } catch (_) {}
  }

  Future<void> _onTokenRefresh(String newToken) async {
    try {
      await ApiService().putFCMToken(newToken);
    } catch (_) {}
  }
}

/// Background message handler — must be a top-level function, not a method.
/// Flutter executes this in a separate isolate when the app is terminated or
/// in the background. It must initialize its own FlutterLocalNotificationsPlugin
/// instance since it cannot access the FCMService singleton from the main isolate.
@pragma('vm:entry-point')
Future<void> firebaseMessagingBackgroundHandler(RemoteMessage message) async {
  await Firebase.initializeApp();

  final String? title = message.data['title'] as String?;
  final String? body = message.data['body'] as String?;
  if (title == null && body == null) return;

  final FlutterLocalNotificationsPlugin localNotifications =
      FlutterLocalNotificationsPlugin();

  const AndroidInitializationSettings androidSettings =
      AndroidInitializationSettings('@mipmap/ic_launcher');
  await localNotifications.initialize(
    settings: const InitializationSettings(android: androidSettings),
  );

  await localNotifications
      .resolvePlatformSpecificImplementation<
          AndroidFlutterLocalNotificationsPlugin>()
      ?.createNotificationChannel(
        const AndroidNotificationChannel(
          _notificationChannelId,
          _notificationChannelName,
          description: _notificationChannelDescription,
          importance: Importance.max,
        ),
      );

  await localNotifications.show(
    id: message.hashCode,
    title: title,
    body: body,
    notificationDetails: NotificationDetails(
      android: AndroidNotificationDetails(
        _notificationChannelId,
        _notificationChannelName,
        channelDescription: _notificationChannelDescription,
        importance: Importance.max,
        priority: Priority.max,
        playSound: true,
        enableVibration: true,
        visibility: NotificationVisibility.public,
      ),
    ),
    payload: jsonEncode(message.data),
  );
}
