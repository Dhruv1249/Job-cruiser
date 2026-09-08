import "package:flutter/material.dart";
import "package:flutter_dotenv/flutter_dotenv.dart";
import "package:flutter_test/flutter_test.dart";
import "package:flutter_app/preferences.dart";
import "package:flutter_app/profile.dart";
import "package:flutter_app/screens/edit_profile_screen.dart";

void main() {
  setUpAll(() {
    dotenv.loadFromString(envString: "API_BASE_URL=http://localhost:8080");
  });

  group("Profile Structured Details and Preferences Separation Tests", () {
    final mockProfileData = {
      "full_name": "Dhruv Dev",
      "primary_email": "dhruv@example.com",
      "phone": "+91 9876543210",
      "location": "Bengaluru, India",
      "avatar_url": null,
    };

    final mockPreferencesData = {
      "full_name": "Dhruv Dev",
      "email": "dhruv@example.com",
      "phone": "+91 9876543210",
      "location": "Bengaluru, India",
      "linkedin_url": "https://linkedin.com/in/dhruvdev",
      "github_url": "https://github.com/dhruvdev",
      "portfolio_url": "https://dhruvdev.com",
      "bio_summary": "Systems engineer specializing in Go and Flutter.",
      "skills": ["Go", "Flutter", "PostgreSQL"],
      "projects": [
        {
          "title": "Job Cruiser Platform",
          "description": "Autonomous AI-driven job searching ecosystem.",
          "tech_stack": ["Go", "Flutter"],
          "link": "https://github.com/example/job-cruiser",
        },
      ],
      "experiences": [
        {
          "company": "Tech Corp",
          "role": "Lead Architect",
          "duration": "2023 - Present",
          "highlights": "Designed high-performance microservices.",
        },
      ],
      "education": [
        {
          "institution": "National Institute of Technology",
          "degree": "B.Tech Computer Science",
          "year": "2019 - 2023",
          "grade": "9.1 CGPA",
        },
      ],
      "target_roles": ["Backend Engineer", "Full Stack Developer"],
      "target_industries": ["Tech", "Fintech"],
      "min_salary": 120000,
    };

    testWidgets("ProfilePage remains unbloated, showing header, preferences, documents and actions", (tester) async {
      tester.view.physicalSize = const Size(1200, 3200);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      await tester.pumpWidget(
        MaterialApp(
          home: ProfilePage(
            initialProfileData: mockProfileData,
            initialPreferencesData: mockPreferencesData,
          ),
        ),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 200));

      expect(find.text("Dhruv Dev"), findsWidgets);
      expect(find.text("dhruv@example.com"), findsWidgets);
      expect(find.text("JOB PREFERENCES & TARGETS"), findsOneWidget);
      expect(find.text("DOCUMENTS & TAILORING"), findsOneWidget);

      expect(find.text("CONTACT & SOCIAL LINKS"), findsNothing);
      expect(find.text("PROFESSIONAL SUMMARY"), findsNothing);
      expect(find.text("TECHNICAL SKILLS"), findsNothing);
      expect(find.text("FEATURED PROJECTS"), findsNothing);
      expect(find.text("WORK EXPERIENCE"), findsNothing);
      expect(find.text("EDUCATION & CREDENTIALS"), findsNothing);

      expect(find.widgetWithText(ElevatedButton, "Edit Profile & Bio"), findsOneWidget);
      expect(find.widgetWithText(OutlinedButton, "Job Preferences"), findsOneWidget);
    });

    testWidgets("ProfilePage navigation to EditProfileScreen on button tap", (tester) async {
      tester.view.physicalSize = const Size(1200, 2400);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      await tester.pumpWidget(
        MaterialApp(
          home: ProfilePage(
            initialProfileData: mockProfileData,
            initialPreferencesData: mockPreferencesData,
          ),
        ),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 200));

      final editProfileButtonFinder = find.widgetWithText(ElevatedButton, "Edit Profile & Bio");
      expect(editProfileButtonFinder, findsOneWidget);

      await tester.tap(editProfileButtonFinder);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 500));

      expect(find.byType(EditProfileScreen), findsOneWidget);
      expect(find.text("Edit Profile & Experience"), findsOneWidget);
    });

    testWidgets("SetPreferencesScreen renders banner linking to Edit Profile", (tester) async {
      tester.view.physicalSize = const Size(1200, 2400);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(() => tester.view.resetPhysicalSize());

      await tester.pumpWidget(
        const MaterialApp(
          home: SetPreferencesScreen(),
        ),
      );

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 200));

      expect(
        find.text("Looking to update your contact links, bio, skills, or projects?"),
        findsOneWidget,
      );
      expect(find.widgetWithText(TextButton, "Edit Profile"), findsOneWidget);
    });
  });
}
