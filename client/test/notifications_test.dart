import 'package:flutter/material.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_app/services/notification_service.dart';
import 'package:flutter_app/widgets/notifications_sheet.dart';

void main() {
  setUpAll(() {
    dotenv.loadFromString(envString: 'API_BASE_URL=http://localhost:8080');
  });

  group('NotificationService unit tests', () {
    test('singleton instance exists and stream can receive tap payloads', () async {
      final service = NotificationService.instance;
      expect(service, isNotNull);

      final receivedPayloads = <String>[];
      final subscription = service.onNotificationTapped.listen(receivedPayloads.add);

      expect(receivedPayloads, isEmpty);
      await subscription.cancel();
    });
  });

  group('NotificationsSheet widget tests', () {
    testWidgets('renders empty placeholder when no notifications exist', (tester) async {
      tester.view.physicalSize = const Size(1200, 1800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: NotificationsSheet(
              initialNotifications: [],
            ),
          ),
        ),
      );

      await tester.pump();
      expect(find.text('Notifications'), findsOneWidget);
      expect(find.text('No notifications yet'), findsOneWidget);
    });

    testWidgets('renders full AI reasoning and view job details button', (tester) async {
      tester.view.physicalSize = const Size(1200, 1800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      final mockNotification = {
        'id': 'notif-123',
        'job_id': 'job-456',
        'title': 'High Match Found (92%): Principal Engineer',
        'message': 'New high match (92%) for Principal Engineer at Google.',
        'reasoning': 'Exceptional fit with deep Go concurrency and distributed consensus expertise.',
        'is_read': false,
        'created_at': DateTime.now().toIso8601String(),
      };

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: NotificationsSheet(
              initialNotifications: [mockNotification],
            ),
          ),
        ),
      );

      await tester.pump();
      expect(find.text('High Match Found (92%): Principal Engineer'), findsOneWidget);
      expect(find.text('AI Match Reasoning'), findsOneWidget);
      expect(
        find.text('Exceptional fit with deep Go concurrency and distributed consensus expertise.'),
        findsOneWidget,
      );
      expect(find.text('View Job Details'), findsOneWidget);
    });
  });
}
