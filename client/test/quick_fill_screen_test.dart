import "package:flutter/material.dart";
import "package:flutter/services.dart";
import "package:flutter_dotenv/flutter_dotenv.dart";
import "package:flutter_test/flutter_test.dart";
import "package:flutter_app/models/quick_fill_item.dart";
import "package:flutter_app/screens/quick_fill_screen.dart";

void main() {
  setUpAll(() {
    dotenv.loadFromString(envString: "API_BASE_URL=http://localhost:8080");
  });

  group("QuickFillItem Model Tests", () {
    test("serializes and deserializes properly", () {
      const originalItem = QuickFillItem(
        id: "test_custom_1",
        label: "Security Clearance",
        value: "Top Secret",
        category: "Work Authorization",
        isCustom: true,
        isMultiLine: false,
      );

      final serializedMap = originalItem.toJson();
      expect(serializedMap["id"], equals("test_custom_1"));
      expect(serializedMap["label"], equals("Security Clearance"));
      expect(serializedMap["value"], equals("Top Secret"));
      expect(serializedMap["category"], equals("Work Authorization"));
      expect(serializedMap["is_custom"], isTrue);

      final deserializedItem = QuickFillItem.fromJson(serializedMap);
      expect(deserializedItem.id, equals(originalItem.id));
      expect(deserializedItem.label, equals(originalItem.label));
      expect(deserializedItem.value, equals(originalItem.value));
      expect(deserializedItem.category, equals(originalItem.category));
      expect(deserializedItem.isCustom, isTrue);
    });

    test("copyWith produces expected updated instance", () {
      const originalItem = QuickFillItem(
        id: "item_1",
        label: "Notice Period",
        value: "30 Days",
        category: "Work Authorization",
      );

      final updatedItem = originalItem.copyWith(value: "Immediate");
      expect(updatedItem.id, equals("item_1"));
      expect(updatedItem.label, equals("Notice Period"));
      expect(updatedItem.value, equals("Immediate"));
      expect(updatedItem.category, equals("Work Authorization"));
    });
  });

  group("QuickFillScreen Widget Tests", () {
    final Map<String, dynamic> mockProfilePreferences = {
      "full_name": "Dhruv Sharma",
      "email": "dhruv.sharma@example.com",
      "phone": "+91 9876543210",
      "location": "Bengaluru, India",
      "country": "India",
      "linkedin_url": "https://linkedin.com/in/dhruvsharma",
      "github_url": "https://github.com/dhruvsharma",
      "portfolio_url": "https://dhruvsharma.dev",
      "custom_links": [
        {
          "label": "Twitter / X",
          "url": "https://x.com/dhruvsharma",
        }
      ],
      "bio_experience_text": "Experienced software engineer specializing in scalable distributed backends.",
      "skills": ["Go", "Flutter", "PostgreSQL", "Docker", "Kubernetes", "Redis", "Kafka"],
      "experiences": [
        {
          "company": "Cruiser Tech",
          "role": "Staff Software Engineer",
          "duration": "2023 - Present",
          "highlights": "Architected low-latency microservices.",
        }
      ],
      "education": [
        {
          "institution": "Indian Institute of Technology",
          "degree": "B.Tech Computer Science",
          "year": "2021",
          "grade": "9.4 CGPA",
        }
      ],
      "custom_form_answers": {
        "work_authorization": "Authorized to work in India and US",
        "visa_sponsorship": "No sponsorship required",
        "notice_period": "Immediate (0 days)",
        "custom_fields": [
          {
            "id": "custom_field_veteran",
            "label": "Veteran Status",
            "value": "I am not a protected veteran",
            "category": "Work Authorization",
            "is_custom": true,
            "is_multiline": false,
          }
        ],
      },
    };

    testWidgets("renders profile items and standard ATS answers", (tester) async {
      tester.view.physicalSize = const Size(1280, 1600);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: QuickFillScreen(
              initialProfileData: mockProfilePreferences,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text("Quick Fill Vault"), findsOneWidget);
      expect(find.text("Dhruv Sharma"), findsWidgets);
      expect(find.text("dhruv.sharma@example.com"), findsOneWidget);
      expect(find.text("+91 9876543210"), findsOneWidget);
      expect(find.text("Bengaluru, India"), findsOneWidget);
      expect(find.text("https://linkedin.com/in/dhruvsharma"), findsOneWidget);
      expect(find.text("https://github.com/dhruvsharma"), findsOneWidget);
      expect(find.text("Authorized to work in India and US"), findsOneWidget);
      expect(find.text("No sponsorship required"), findsOneWidget);
      expect(find.text("Immediate (0 days)"), findsOneWidget);
      expect(find.text("Veteran Status"), findsOneWidget);
      expect(find.text("I am not a protected veteran"), findsOneWidget);
    });

    testWidgets("filters items by search query", (tester) async {
      tester.view.physicalSize = const Size(1280, 1600);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: QuickFillScreen(
              initialProfileData: mockProfilePreferences,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      final searchTextFieldFinder = find.byType(TextField).first;
      await tester.enterText(searchTextFieldFinder, "github");
      await tester.pumpAndSettle();

      expect(find.text("https://github.com/dhruvsharma"), findsOneWidget);
      expect(find.text("dhruv.sharma@example.com"), findsNothing);
    });

    testWidgets("filters items by category chip selection", (tester) async {
      tester.view.physicalSize = const Size(1280, 1600);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: QuickFillScreen(
              initialProfileData: mockProfilePreferences,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      final socialsCategoryChipFinder = find.widgetWithText(ChoiceChip, "Socials & Links");
      expect(socialsCategoryChipFinder, findsOneWidget);
      await tester.tap(socialsCategoryChipFinder);
      await tester.pumpAndSettle();

      expect(find.text("https://linkedin.com/in/dhruvsharma"), findsOneWidget);
      expect(find.text("https://github.com/dhruvsharma"), findsOneWidget);
      expect(find.text("https://dhruvsharma.dev"), findsOneWidget);
      expect(find.text("dhruv.sharma@example.com"), findsNothing);
    });

    testWidgets("triggers clipboard copy on tap", (tester) async {
      tester.view.physicalSize = const Size(1280, 1600);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      final List<MethodCall> methodChannelCalls = <MethodCall>[];
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(SystemChannels.platform, (MethodCall methodCall) async {
        methodChannelCalls.add(methodCall);
        return null;
      });

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: QuickFillScreen(
              initialProfileData: mockProfilePreferences,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      final emailTileFinder = find.widgetWithText(InkWell, "dhruv.sharma@example.com");
      expect(emailTileFinder, findsOneWidget);
      await tester.tap(emailTileFinder);
      await tester.pump();

      expect(
        methodChannelCalls.any(
          (call) => call.method == "Clipboard.setData" && (call.arguments as Map)["text"] == "dhruv.sharma@example.com",
        ),
        isTrue,
      );
      expect(find.text("Copied"), findsOneWidget);
      expect(find.byIcon(Icons.check), findsOneWidget);
    });

    testWidgets("opens add custom field modal dialog", (tester) async {
      tester.view.physicalSize = const Size(1280, 1600);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: QuickFillScreen(
              initialProfileData: mockProfilePreferences,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      final addFieldButtonFinder = find.widgetWithText(OutlinedButton, "Add Custom Field");
      expect(addFieldButtonFinder, findsOneWidget);
      await tester.tap(addFieldButtonFinder);
      await tester.pumpAndSettle();

      expect(find.text("Field Label *"), findsOneWidget);
      expect(find.text("Value to Copy *"), findsOneWidget);
    });

    testWidgets("displays More button on long content and opens view dialog", (tester) async {
      tester.view.physicalSize = const Size(1280, 2400);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      await tester.pumpWidget(
        MaterialApp(
          home: QuickFillScreen(
            initialProfileData: mockProfilePreferences,
            onSavePreferences: (_) async => true,
          ),
        ),
      );
      await tester.pumpAndSettle();

      final moreButtonFinder = find.text("More");
      expect(moreButtonFinder, findsWidgets);

      await tester.ensureVisible(moreButtonFinder.first);
      await tester.tap(moreButtonFinder.first);
      await tester.pumpAndSettle();

      expect(find.text("Close"), findsOneWidget);
      expect(find.text("Copy Value"), findsOneWidget);
    });

    testWidgets("allows editing and deleting any field including prefilled ones", (tester) async {
      tester.view.physicalSize = const Size(1280, 2400);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      await tester.pumpWidget(
        MaterialApp(
          home: QuickFillScreen(
            initialProfileData: mockProfilePreferences,
            onSavePreferences: (_) async => true,
          ),
        ),
      );
      await tester.pumpAndSettle();

      final editIconButtons = find.byIcon(Icons.edit_outlined);
      expect(editIconButtons, findsWidgets);

      await tester.tap(editIconButtons.first);
      await tester.pumpAndSettle();

      expect(find.text("Cancel"), findsOneWidget);
      expect(find.text("Save"), findsOneWidget);

      final valueFieldFinder = find.widgetWithText(TextField, "Dhruv Sharma");
      expect(valueFieldFinder, findsOneWidget);
      await tester.enterText(valueFieldFinder, "Dhruv Sharma Senior");
      FocusManager.instance.primaryFocus?.unfocus();
      await tester.pump();
      await tester.tap(find.text("Save"));
      await tester.pumpAndSettle();

      expect(find.text("Dhruv Sharma Senior"), findsWidgets);

      final deleteIconButtons = find.byIcon(Icons.delete_outline);
      expect(deleteIconButtons, findsWidgets);

      await tester.tap(deleteIconButtons.first);
      await tester.pumpAndSettle();

      expect(find.text("Delete Field"), findsOneWidget);
      await tester.tap(find.text("Delete"));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));

      expect(find.text("Dhruv Sharma Senior"), findsNothing);
      expect(find.text("Undo"), findsOneWidget);

      await tester.tap(find.text("Undo"));
      await tester.pumpAndSettle();

      expect(find.text("Dhruv Sharma"), findsWidgets);
    });
  });
}
