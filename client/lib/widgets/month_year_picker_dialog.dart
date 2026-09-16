import "package:flutter/material.dart";

/// Normalizes and formats a date range. If start and end are in the same month and year, formats as a single month.
String formatCustomDateRange(String start, String end) {
  final cleanStart = start.trim();
  final cleanEnd = end.trim();
  if (cleanStart.isEmpty && cleanEnd.isEmpty) {
    return "";
  }
  if (cleanStart.isEmpty) {
    return cleanEnd;
  }
  if (cleanEnd.isEmpty) {
    return cleanStart;
  }

  final normalizedStart = cleanStart.replaceAll(",", "").replaceAll(RegExp(r"\s+"), " ").toLowerCase();
  final normalizedEnd = cleanEnd.replaceAll(",", "").replaceAll(RegExp(r"\s+"), " ").toLowerCase();

  if (normalizedStart == normalizedEnd) {
    return cleanStart;
  }
  return "$cleanStart - $cleanEnd";
}

/// Normalizes any duration string so that duplicate ranges like 'Nov, 2025 - Nov, 2025' or 'Nov 2025 - Nov 2025' collapse to 'Nov 2025'.
String formatDisplayDuration(String rawDuration) {
  final trimmed = rawDuration.trim();
  if (trimmed.isEmpty) {
    return "";
  }

  final parts = trimmed.split(RegExp(r"\s*(?:-|–|—|to)\s*"));
  if (parts.length == 2) {
    final startPart = parts[0].trim();
    final endPart = parts[1].trim();
    final normalizedStart = startPart.replaceAll(",", "").replaceAll(RegExp(r"\s+"), " ").toLowerCase();
    final normalizedEnd = endPart.replaceAll(",", "").replaceAll(RegExp(r"\s+"), " ").toLowerCase();
    if (normalizedStart.isNotEmpty && normalizedStart == normalizedEnd) {
      return startPart;
    }
  }
  return trimmed;
}

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
    final cleanedInitial = initialMonthYear.trim().replaceAll(",", "");
    if (cleanedInitial.toLowerCase() == "present") {
      isMarkedPresent = true;
    } else {
      final textParts = cleanedInitial.split(" ");
      if (textParts.isNotEmpty) {
        final candidateMonth = textParts[0].trim();
        final matchedMonth = monthsList.firstWhere(
          (month) => month.toLowerCase() == candidateMonth.toLowerCase() || candidateMonth.toLowerCase().startsWith(month.toLowerCase()),
          orElse: () => "Jan",
        );
        selectedMonth = matchedMonth;
      }
      if (textParts.length >= 2) {
        final parsedYear = int.tryParse(textParts.last.trim());
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
  final Widget? prefixIcon;

  const DateRangePickerField({
    super.key,
    required this.controller,
    this.labelText = "Duration Range",
    this.hintText = "e.g. Nov 2021 - Present",
    this.prefixIcon,
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
                  final textParts = controller.text.split(RegExp(r"\s*(?:-|–|—|to)\s*"));
                  final initialStart = textParts.isNotEmpty && textParts[0].trim().isNotEmpty ? textParts[0].trim() : null;
                  final selectedDate = await showCustomMonthYearPicker(
                    context: context,
                    title: "Select Start Date",
                    initialMonthYear: initialStart,
                    allowPresent: false,
                  );
                  if (selectedDate != null) {
                    final currentEnd = textParts.length > 1 && textParts[1].trim().isNotEmpty
                        ? textParts[1].trim()
                        : (textParts.isNotEmpty && textParts[0].trim().isNotEmpty && textParts[0].trim().toLowerCase() != "present" ? textParts[0].trim() : "Present");
                    controller.text = formatCustomDateRange(selectedDate, currentEnd);
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
                  final textParts = controller.text.split(RegExp(r"\s*(?:-|–|—|to)\s*"));
                  final initialEnd = textParts.length > 1 && textParts[1].trim().isNotEmpty
                      ? textParts[1].trim()
                      : (textParts.isNotEmpty && textParts[0].trim().isNotEmpty ? textParts[0].trim() : null);
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
                    controller.text = formatCustomDateRange(currentStart, selectedDate);
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
          minLines: 1,
          maxLines: 1,
          decoration: InputDecoration(
            labelText: labelText,
            hintText: hintText,
            prefixIcon: prefixIcon,
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
  final Widget? prefixIcon;

  const SingleDatePickerField({
    super.key,
    required this.controller,
    required this.labelText,
    this.hintText = "e.g. Oct 2024 or Present",
    this.allowPresent = false,
    this.prefixIcon,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Expanded(
          child: TextField(
            controller: controller,
            minLines: 1,
            maxLines: 1,
            decoration: InputDecoration(
              labelText: labelText,
              hintText: hintText,
              prefixIcon: prefixIcon,
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
