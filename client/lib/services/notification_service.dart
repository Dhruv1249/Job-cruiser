import 'dart:async';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';

/// NotificationService manages system and heads-up notifications on mobile devices.
class NotificationService {
  NotificationService._internal();

  static final NotificationService instance = NotificationService._internal();

  final FlutterLocalNotificationsPlugin _notificationsPlugin = FlutterLocalNotificationsPlugin();
  final StreamController<String> _payloadStreamController = StreamController<String>.broadcast();
  bool _isInitialized = false;

  static const String _channelId = 'job_cruiser_tailor_channel';
  static const String _channelName = 'Job Cruiser Alerts';
  static const String _channelDescription = 'Notifications for resume tailoring, cover letters, and match updates';

  /// Stream of notification payloads tapped by the user.
  Stream<String> get onNotificationTapped => _payloadStreamController.stream;

  /// Initializes the local notification plugin with platform settings.
  Future<void> initialize() async {
    if (_isInitialized) return;

    const androidInitialization = AndroidInitializationSettings('@mipmap/launcher_icon');
    const darwinInitialization = DarwinInitializationSettings(
      requestAlertPermission: true,
      requestBadgePermission: true,
      requestSoundPermission: true,
    );

    const initializationSettings = InitializationSettings(
      android: androidInitialization,
      iOS: darwinInitialization,
      macOS: darwinInitialization,
    );

    await _notificationsPlugin.initialize(
      settings: initializationSettings,
      onDidReceiveNotificationResponse: (response) {
        final payload = response.payload;
        if (payload != null && payload.isNotEmpty) {
          _payloadStreamController.add(payload);
        }
      },
    );

    final launchDetails = await _notificationsPlugin.getNotificationAppLaunchDetails();
    if (launchDetails != null && launchDetails.didNotificationLaunchApp) {
      final payload = launchDetails.notificationResponse?.payload;
      if (payload != null && payload.isNotEmpty) {
        _payloadStreamController.add(payload);
      }
    }

    final androidImplementation =
        _notificationsPlugin.resolvePlatformSpecificImplementation<AndroidFlutterLocalNotificationsPlugin>();
    if (androidImplementation != null) {
      const channel = AndroidNotificationChannel(
        _channelId,
        _channelName,
        description: _channelDescription,
        importance: Importance.high,
        playSound: true,
      );
      await androidImplementation.createNotificationChannel(channel);
      await androidImplementation.requestNotificationsPermission();
    }

    _isInitialized = true;
  }

  /// Displays an immediate system notification on the user's mobile device with full AI reasoning.
  Future<void> showLocalNotification({
    int id = 0,
    required String title,
    required String body,
    String? payload,
  }) async {
    if (!_isInitialized) {
      await initialize();
    }

    final androidNotificationDetails = AndroidNotificationDetails(
      _channelId,
      _channelName,
      channelDescription: _channelDescription,
      importance: Importance.high,
      priority: Priority.high,
      playSound: true,
      icon: '@mipmap/launcher_icon',
      styleInformation: BigTextStyleInformation(
        body,
        contentTitle: title,
        summaryText: 'Job Cruiser Match Alert',
      ),
    );

    const darwinNotificationDetails = DarwinNotificationDetails(
      presentAlert: true,
      presentBadge: true,
      presentSound: true,
    );

    final notificationDetails = NotificationDetails(
      android: androidNotificationDetails,
      iOS: darwinNotificationDetails,
      macOS: darwinNotificationDetails,
    );

    await _notificationsPlugin.show(
      id: id,
      title: title,
      body: body,
      notificationDetails: notificationDetails,
      payload: payload,
    );
  }
}
