import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('Duplicate inbox tap trigger increments value and fires registered listener', () {
    final refreshNotifier = ValueNotifier<int>(0);
    var refreshCallCount = 0;

    void onRefresh() {
      refreshCallCount++;
    }

    refreshNotifier.addListener(onRefresh);

    final currentIndex = 0;
    final tappedIndex = 0;

    if (currentIndex == tappedIndex && tappedIndex == 0) {
      refreshNotifier.value++;
    }

    expect(refreshCallCount, equals(1));
    expect(refreshNotifier.value, equals(1));

    if (currentIndex == tappedIndex && tappedIndex == 0) {
      refreshNotifier.value++;
    }

    expect(refreshCallCount, equals(2));
    expect(refreshNotifier.value, equals(2));

    refreshNotifier.removeListener(onRefresh);
    refreshNotifier.dispose();
  });

  test('Switching from other tab to inbox does not trigger duplicate refresh', () {
    final refreshNotifier = ValueNotifier<int>(0);
    var refreshCallCount = 0;

    refreshNotifier.addListener(() {
      refreshCallCount++;
    });

    var activeIndex = 1;
    final targetIndex = 0;

    if (activeIndex == targetIndex && targetIndex == 0) {
      refreshNotifier.value++;
    } else {
      activeIndex = targetIndex;
    }

    expect(refreshCallCount, equals(0));
    expect(activeIndex, equals(0));

    refreshNotifier.dispose();
  });
}
