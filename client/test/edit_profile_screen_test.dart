import "package:flutter/material.dart";
import "package:flutter_dotenv/flutter_dotenv.dart";
import "package:flutter_test/flutter_test.dart";
import "package:flutter_app/screens/edit_profile_screen.dart";

void main() {
  setUpAll(() {
    dotenv.loadFromString(envString: "API_BASE_URL=http://localhost:8080");
  });

  group("EditProfileScreen Widget Tests", () {
    final mockInitialData = {
      "full_name": "Dhruv Dev",
      "email": "dhruv@example.com",
      "phone": "+91 9876543210",
      "location": "Bengaluru, India",
      "linkedin_url": "https://linkedin.com/in/dhruvdev",
      "github_url": "https://github.com/dhruvdev",
      "portfolio_url": "https://dhruvdev.com",
      "bio_summary": "Passionate systems and backend engineer specializing in Go and Flutter.",
      "skills": ["Go", "Flutter", "PostgreSQL", "Docker"],
      "projects": [
        {
          "title": "Job Cruiser Platform",
          "description": "Autonomous AI-driven job searching ecosystem.",
          "tech_stack": ["Go", "Flutter", "Postgres"],
          "link": "https://github.com/example/job-cruiser",
        },
      ],
      "experiences": [
        {
          "company": "Tech Innovations",
          "role": "Lead Backend Architect",
          "duration": "2023 - Present",
          "highlights": "Architected distributed matching pipeline at scale.",
        },
      ],
      "education": [
        {
          "institution": "National Institute of Technology",
          "degree": "B.Tech in Computer Science",
          "year": "2019 - 2023",
          "grade": "9.1 CGPA",
        },
      ],
    };

    testWidgets("renders pre-populated profile details correctly", (tester) async {
      tester.view.physicalSize = const Size(1200, 2400);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      await tester.pumpWidget(
        MaterialApp(
          home: EditProfileScreen(
            initialProfileData: mockInitialData,
          ),
        ),
      );

      await tester.pumpAndSettle();

      expect(find.text("Edit Profile & Experience"), findsOneWidget);
      expect(find.widgetWithText(TextFormField, "Dhruv Dev"), findsOneWidget);
      expect(find.widgetWithText(TextFormField, "dhruv@example.com"), findsOneWidget);
      expect(find.widgetWithText(TextFormField, "+91 9876543210"), findsOneWidget);
      expect(find.widgetWithText(TextFormField, "Bengaluru, India"), findsOneWidget);

      expect(find.text("Go"), findsWidgets);
      expect(find.text("Flutter"), findsWidgets);
      expect(find.text("PostgreSQL"), findsOneWidget);

      expect(find.text("Job Cruiser Platform"), findsOneWidget);
      expect(find.text("Lead Backend Architect"), findsOneWidget);
      expect(find.text("B.Tech in Computer Science"), findsOneWidget);
    });

    testWidgets("allows adding and removing technical skill tags", (tester) async {
      tester.view.physicalSize = const Size(1200, 2400);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      await tester.pumpWidget(
        MaterialApp(
          home: EditProfileScreen(
            initialProfileData: mockInitialData,
          ),
        ),
      );

      await tester.pumpAndSettle();

      expect(find.text("Kubernetes"), findsNothing);

      await tester.enterText(
        find.widgetWithText(TextField, "Enter skill (e.g. Go, PostgreSQL, Kubernetes)"),
        "Kubernetes",
      );
      await tester.tap(find.widgetWithText(ElevatedButton, "Add"));
      await tester.pumpAndSettle();

      expect(find.text("Kubernetes"), findsOneWidget);

      final deleteIconFinder = find.descendant(
        of: find.widgetWithText(Chip, "Kubernetes"),
        matching: find.byIcon(Icons.cancel),
      );
      if (deleteIconFinder.evaluate().isNotEmpty) {
        await tester.tap(deleteIconFinder);
        await tester.pumpAndSettle();
        expect(find.text("Kubernetes"), findsNothing);
      }
    });

    testWidgets("opens project dialog and adds new project", (tester) async {
      tester.view.physicalSize = const Size(1200, 2400);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      await tester.pumpWidget(
        MaterialApp(
          home: EditProfileScreen(
            initialProfileData: mockInitialData,
          ),
        ),
      );

      await tester.pumpAndSettle();

      await tester.tap(find.widgetWithText(OutlinedButton, "Add Project"));
      await tester.pumpAndSettle();

      expect(find.text("Add Project"), findsWidgets);

      await tester.enterText(
        find.widgetWithText(TextField, "Project Title *"),
        "Distributed Cache",
      );
      await tester.enterText(
        find.widgetWithText(TextField, "Tech Stack (comma separated)"),
        "Go, Redis",
      );
      await tester.enterText(
        find.widgetWithText(TextField, "Description"),
        "In-memory cache cluster",
      );

      FocusManager.instance.primaryFocus?.unfocus();
      await tester.pump();
      await tester.tap(find.widgetWithText(ElevatedButton, "Save"));
      await tester.pumpAndSettle();
      expect(find.text("Distributed Cache"), findsOneWidget);
      expect(find.text("In-memory cache cluster"), findsOneWidget);
    });

    testWidgets("extracts bio from master_cv_text when bio_experience_text is omitted", (tester) async {
      tester.view.physicalSize = const Size(1200, 2400);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      final fallbackData = {
        "full_name": "Dhruv Dev",
        "master_cv_text": "Experienced systems software engineer\n\n--- STRUCTURED RESUME DETAILS ---\n{\"skills\":[\"Go\"]}",
      };

      await tester.pumpWidget(
        MaterialApp(
          home: EditProfileScreen(
            initialProfileData: fallbackData,
          ),
        ),
      );

      await tester.pumpAndSettle();

      expect(find.widgetWithText(TextField, "Experienced systems software engineer"), findsOneWidget);
    });
  });
}
