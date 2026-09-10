import "package:flutter/material.dart";

/// Shows a dialog allowing the candidate to select a month and year or mark an entry as Present.
Future<String?> showCustomMonthYearPicker({
  required BuildContext context,
  required String title,
  String? initialMonthYear,
  bool allowPresent = false,
}) async {
  final monthsList = [
    "Jan",
    "Feb",
    "Mar",
    "Apr",
    "May",
    "Jun",
    "Jul",
    "Aug",
    "Sep",
    "Oct",
    "Nov",
    "Dec",
  ];
  final currentYearValue = DateTime.now().year;
  final yearsList = List<int>.generate(40, (index) => currentYearValue - 30 + index);

  String selectedMonth = "Jan";
  int selectedYear = currentYearValue;
  bool isMarkedPresent = false;

  if (initialMonthYear != null && initialMonthYear.trim().isNotEmpty) {
    if (initialMonthYear.trim().toLowerCase() == "present") {
      isMarkedPresent = true;
    } else {
      final textParts = initialMonthYear.trim().split(" ");
      if (textParts.isNotEmpty && monthsList.contains(textParts[0])) {
        selectedMonth = textParts[0];
      }
      if (textParts.length >= 2) {
        final parsedYear = int.tryParse(textParts[1]);
        if (parsedYear != null && yearsList.contains(parsedYear)) {
          selectedYear = parsedYear;
        }
      }
    }
  }

  return showDialog<String>(
    context: context,
    builder: (dialogContext) {
      return StatefulBuilder(
        builder: (dialogContext, setDialogState) {
          return AlertDialog(
            title: Text(title),
            content: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (allowPresent) ...[
                  CheckboxListTile(
                    title: const Text("Present (Ongoing / Currently Active)"),
                    value: isMarkedPresent,
                    onChanged: (newValue) {
                      setDialogState(() {
                        isMarkedPresent = newValue ?? false;
                      });
                    },
                    controlAffinity: ListTileControlAffinity.leading,
                    contentPadding: EdgeInsets.zero,
                  ),
                  const Divider(),
                ],
                if (!isMarkedPresent) ...[
                  Row(
                    children: [
                      Expanded(
                        child: DropdownButtonFormField<String>(
                          initialValue: selectedMonth,
                          decoration: const InputDecoration(
                            labelText: "Month",
                            border: OutlineInputBorder(),
                            isDense: true,
                          ),
                          items: monthsList
                              .map((monthName) => DropdownMenuItem(
                                    value: monthName,
                                    child: Text(monthName),
                                  ))
                              .toList(),
                          onChanged: (updatedMonth) {
                            if (updatedMonth != null) {
                              setDialogState(() {
                                selectedMonth = updatedMonth;
                              });
                            }
                          },
                        ),
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                        child: DropdownButtonFormField<int>(
                          initialValue: selectedYear,
                          decoration: const InputDecoration(
                            labelText: "Year",
                            border: OutlineInputBorder(),
                            isDense: true,
                          ),
                          items: yearsList
                              .map((yearValue) => DropdownMenuItem(
                                    value: yearValue,
                                    child: Text("$yearValue"),
                                  ))
                              .toList(),
                          onChanged: (updatedYear) {
                            if (updatedYear != null) {
                              setDialogState(() {
                                selectedYear = updatedYear;
                              });
                            }
                          },
                        ),
                      ),
                    ],
                  ),
                ],
              ],
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(dialogContext).pop(),
                child: const Text("Cancel"),
              ),
              ElevatedButton(
                onPressed: () {
                  if (isMarkedPresent) {
                    Navigator.of(dialogContext).pop("Present");
                  } else {
                    Navigator.of(dialogContext).pop("$selectedMonth $selectedYear");
                  }
                },
                child: const Text("Select"),
              ),
            ],
          );
        },
      );
    },
  );
}

/// Renders a date-range input field with quick start and end date picker actions including Present toggle.
class DateRangePickerField extends StatelessWidget {
  final TextEditingController controller;
  final String labelText;
  final String hintText;

  const DateRangePickerField({
    super.key,
    required this.controller,
    this.labelText = "Duration Range",
    this.hintText = "e.g. Nov 2021 - Present",
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Expanded(
              child: OutlinedButton.icon(
                onPressed: () async {
                  final textParts = controller.text.split(" - ");
                  final initialStart = textParts.isNotEmpty ? textParts[0].trim() : null;
                  final selectedDate = await showCustomMonthYearPicker(
                    context: context,
                    title: "Select Start Date",
                    initialMonthYear: initialStart,
                    allowPresent: false,
                  );
                  if (selectedDate != null) {
                    final currentEnd = textParts.length > 1 && textParts[1].trim().isNotEmpty
                        ? textParts[1].trim()
                        : "Present";
                    controller.text = "$selectedDate - $currentEnd";
                  }
                },
                icon: const Icon(Icons.calendar_month, size: 16),
                label: const Text("Start Date"),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: OutlinedButton.icon(
                onPressed: () async {
                  final textParts = controller.text.split(" - ");
                  final initialEnd = textParts.length > 1 ? textParts[1].trim() : null;
                  final selectedDate = await showCustomMonthYearPicker(
                    context: context,
                    title: "Select End Date",
                    initialMonthYear: initialEnd,
                    allowPresent: true,
                  );
                  if (selectedDate != null) {
                    final currentStart = textParts.isNotEmpty && textParts[0].trim().isNotEmpty
                        ? textParts[0].trim()
                        : "Jan 2023";
                    controller.text = "$currentStart - $selectedDate";
                  }
                },
                icon: const Icon(Icons.event_available, size: 16),
                label: const Text("End Date"),
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        TextField(
          controller: controller,
          decoration: InputDecoration(
            labelText: labelText,
            hintText: hintText,
            isDense: true,
            suffixIcon: controller.text.isNotEmpty
                ? IconButton(
                    icon: const Icon(Icons.clear, size: 18),
                    onPressed: () {
                      controller.clear();
                    },
                  )
                : null,
          ),
        ),
      ],
    );
  }
}

/// Renders a single-date selector with a quick picker button and direct text field editing.
class SingleDatePickerField extends StatelessWidget {
  final TextEditingController controller;
  final String labelText;
  final String hintText;
  final bool allowPresent;

  const SingleDatePickerField({
    super.key,
    required this.controller,
    required this.labelText,
    this.hintText = "e.g. Oct 2024 or Present",
    this.allowPresent = false,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Expanded(
          child: TextField(
            controller: controller,
            decoration: InputDecoration(
              labelText: labelText,
              hintText: hintText,
              isDense: true,
            ),
          ),
        ),
        const SizedBox(width: 8),
        IconButton.outlined(
          tooltip: "Pick Month & Year",
          icon: const Icon(Icons.calendar_month, size: 20),
          onPressed: () async {
            final selectedDate = await showCustomMonthYearPicker(
              context: context,
              title: "Select $labelText",
              initialMonthYear: controller.text.trim(),
              allowPresent: allowPresent,
            );
            if (selectedDate != null) {
              controller.text = selectedDate;
            }
          },
        ),
      ],
    );
  }
}
