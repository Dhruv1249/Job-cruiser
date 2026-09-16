import "package:flutter/material.dart";
import "package:flutter_test/flutter_test.dart";
import "package:flutter_app/widgets/month_year_picker_dialog.dart";

void main() {
  group("Date Formatting Unit Tests", () {
    test("formatCustomDateRange collapses identical start and end dates", () {
      expect(formatCustomDateRange("Nov 2025", "Nov 2025"), "Nov 2025");
      expect(formatCustomDateRange("Nov, 2025", "Nov 2025"), "Nov, 2025");
      expect(formatCustomDateRange("Oct 2024", "Oct 2024"), "Oct 2024");
      expect(formatCustomDateRange("2023", "2023"), "2023");
    });

    test("formatCustomDateRange preserves distinct start and end dates", () {
      expect(formatCustomDateRange("Jan 2023", "Dec 2024"), "Jan 2023 - Dec 2024");
      expect(formatCustomDateRange("Aug 2020", "May 2024"), "Aug 2020 - May 2024");
      expect(formatCustomDateRange("Jan 2023", "Present"), "Jan 2023 - Present");
    });

    test("formatCustomDateRange handles empty or single inputs", () {
      expect(formatCustomDateRange("", ""), "");
      expect(formatCustomDateRange("Nov 2025", ""), "Nov 2025");
      expect(formatCustomDateRange("", "Nov 2025"), "Nov 2025");
    });

    test("formatDisplayDuration collapses single-month ranges with various delimiters", () {
      expect(formatDisplayDuration("Nov, 2025 - Nov, 2025"), "Nov, 2025");
      expect(formatDisplayDuration("Nov 2025 - Nov 2025"), "Nov 2025");
      expect(formatDisplayDuration("Nov 2025 – Nov 2025"), "Nov 2025");
      expect(formatDisplayDuration("Nov 2025 — Nov 2025"), "Nov 2025");
      expect(formatDisplayDuration("Nov 2025 to Nov 2025"), "Nov 2025");
      expect(formatDisplayDuration("2024 - 2024"), "2024");
    });

    test("formatDisplayDuration preserves multi-month and ongoing ranges", () {
      expect(formatDisplayDuration("Jan 2023 - Dec 2024"), "Jan 2023 - Dec 2024");
      expect(formatDisplayDuration("Oct 2024 - Present"), "Oct 2024 - Present");
      expect(formatDisplayDuration("Nov 2025"), "Nov 2025");
      expect(formatDisplayDuration(""), "");
    });
  });

  group("Date Picker Form Fields Tests", () {
    testWidgets("DateRangePickerField renders and displays initial value", (tester) async {
      final controller = TextEditingController(text: "Nov 2025 - Dec 2025");

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: DateRangePickerField(
              controller: controller,
              labelText: "Study Period",
            ),
          ),
        ),
      );

      expect(find.text("Study Period"), findsOneWidget);
      expect(find.text("Nov 2025 - Dec 2025"), findsOneWidget);
      expect(find.byIcon(Icons.calendar_month), findsOneWidget);
      expect(find.byIcon(Icons.event_available), findsOneWidget);
    });

    testWidgets("SingleDatePickerField renders and displays initial value", (tester) async {
      final controller = TextEditingController(text: "Oct 2024");

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SingleDatePickerField(
              controller: controller,
              labelText: "Date Received",
            ),
          ),
        ),
      );

      expect(find.text("Date Received"), findsOneWidget);
      expect(find.text("Oct 2024"), findsOneWidget);
      expect(find.byIcon(Icons.calendar_month), findsOneWidget);
    });
  });
}
