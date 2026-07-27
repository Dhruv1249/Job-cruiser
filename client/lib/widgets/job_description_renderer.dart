import 'package:flutter/material.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import 'package:html/parser.dart' show parseFragment;
import 'package:url_launcher/url_launcher.dart';
import '../main.dart' show AppColors;

/// Enumeration of detected job description content formats.
enum JobDescFormat {
  html,
  markdown,
  plainText,
}

/// Unescapes HTML entities (such as &nbsp;, &amp;, &lt;, &gt;, &quot;, &#39;) into normal text characters.
String unescapeHtmlEntities(String text) {
  if (text.isEmpty) return text;
  final fragment = parseFragment(text);
  final decoded = fragment.text ?? text;
  return decoded.replaceAll('\u00A0', ' ');
}

/// Automated content format detector for job descriptions.
JobDescFormat detectJobDescFormat(String text) {
  final trimmed = text.trim();
  if (trimmed.isEmpty) return JobDescFormat.plainText;

  final htmlRegex = RegExp(
    r'<[a-z][\s\S]*?>',
    caseSensitive: false,
  );
  if (htmlRegex.hasMatch(trimmed)) {
    return JobDescFormat.html;
  }

  final markdownRegex = RegExp(
    r'(^#+\s|\*\*.*?\*\*|\[.*?\]\(https?://.*?\)|^\s*[-\*]\s|```)',
    multiLine: true,
  );
  if (markdownRegex.hasMatch(trimmed)) {
    return JobDescFormat.markdown;
  }

  return JobDescFormat.plainText;
}

/// Dynamic renderer component that automatically detects content format
/// and renders styled HTML, Markdown, or Plain Text using official HTML and Markdown parsers.
class JobDescriptionRenderer extends StatelessWidget {
  const JobDescriptionRenderer({
    super.key,
    required this.content,
    this.onLinkTap,
  });

  final String content;
  final Function(String url)? onLinkTap;

  Future<void> _handleUrlClick(String url) async {
    if (onLinkTap != null) {
      onLinkTap!(url);
      return;
    }
    final uri = Uri.tryParse(url.trim());
    if (uri != null) {
      try {
        await launchUrl(uri, mode: LaunchMode.externalApplication);
      } catch (_) {}
    }
  }

  @override
  Widget build(BuildContext context) {
    final format = detectJobDescFormat(content);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            const Text(
              'Job Description',
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w700,
                color: AppColors.primary,
              ),
            ),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
              decoration: BoxDecoration(
                color: AppColors.surfaceContainerHigh,
                borderRadius: BorderRadius.circular(4),
              ),
              child: Text(
                format == JobDescFormat.html
                    ? 'HTML'
                    : format == JobDescFormat.markdown
                        ? 'MARKDOWN'
                        : 'TEXT',
                style: const TextStyle(
                  fontSize: 10,
                  fontWeight: FontWeight.w700,
                  color: AppColors.secondary,
                  letterSpacing: 0.5,
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 12),
        _buildContentWidget(context, format),
      ],
    );
  }

  Widget _buildContentWidget(BuildContext context, JobDescFormat format) {
    switch (format) {
      case JobDescFormat.html:
        return _renderHtmlContent(rawHtml: content);
      case JobDescFormat.markdown:
        return _renderMarkdownContent(context: context, rawMd: content);
      case JobDescFormat.plainText:
        return _renderPlainTextContent(unescapeHtmlEntities(content));
    }
  }

  Widget _renderHtmlContent({required String rawHtml}) {
    String formattedHtml = rawHtml
        .replaceAll(RegExp(r'</?(p|div|br\s*/?|h[1-6])>', caseSensitive: false), '\n')
        .replaceAll(RegExp(r'<li>', caseSensitive: false), '\n• ')
        .replaceAll(RegExp(r'</li>', caseSensitive: false), '')
        .replaceAll(RegExp(r'</?(ul|ol)>', caseSensitive: false), '\n')
        .replaceAll(RegExp(r'<[^>]*>'), '');

    final decodedText = unescapeHtmlEntities(formattedHtml);
    final cleanText = decodedText.replaceAll(RegExp(r'\n{3,}'), '\n\n').trim();

    return _renderPlainTextContent(cleanText);
  }

  Widget _renderMarkdownContent({
    required BuildContext context,
    required String rawMd,
  }) {
    final cleanMd = unescapeHtmlEntities(rawMd);

    return MarkdownBody(
      data: cleanMd,
      onTapLink: (text, href, title) {
        if (href != null) {
          _handleUrlClick(href);
        }
      },
      styleSheet: MarkdownStyleSheet.fromTheme(Theme.of(context)).copyWith(
        p: const TextStyle(
          fontSize: 14,
          height: 1.5,
          color: AppColors.onSurface,
        ),
        h1: const TextStyle(
          fontSize: 18,
          fontWeight: FontWeight.w700,
          color: AppColors.primary,
        ),
        h2: const TextStyle(
          fontSize: 16,
          fontWeight: FontWeight.w700,
          color: AppColors.primary,
        ),
        h3: const TextStyle(
          fontSize: 15,
          fontWeight: FontWeight.w600,
          color: AppColors.primary,
        ),
        listBullet: const TextStyle(
          fontSize: 14,
          color: AppColors.onSurface,
        ),
        a: const TextStyle(
          color: AppColors.secondary,
          decoration: TextDecoration.underline,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }

  Widget _renderPlainTextContent(String text) {
    final paragraphs = text.split(RegExp(r'\n\s*\n'));
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: paragraphs.map((para) {
        final lines = para.trim().split('\n');
        return Padding(
          padding: const EdgeInsets.only(bottom: 12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: lines.map((line) {
              final trimmed = line.trim();
              if (trimmed.startsWith('• ') || trimmed.startsWith('- ')) {
                return Padding(
                  padding: const EdgeInsets.only(left: 8, bottom: 4),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text('• ', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 14)),
                      Expanded(child: _buildRichSpanText(trimmed.substring(2))),
                    ],
                  ),
                );
              }
              return Padding(
                padding: const EdgeInsets.only(bottom: 4),
                child: _buildRichSpanText(trimmed),
              );
            }).toList(),
          ),
        );
      }).toList(),
    );
  }

  Widget _buildRichSpanText(String inputStr) {
    final urlRegex = RegExp(r'\[([^\]]+)\]\((https?://[^\)]+)\)|(https?://[^\s\)]+)');
    final List<InlineSpan> spans = [];

    int lastIndex = 0;
    for (final Match match in urlRegex.allMatches(inputStr)) {
      if (match.start > lastIndex) {
        spans.add(TextSpan(text: inputStr.substring(lastIndex, match.start)));
      }

      final String label = match.group(1) ?? match.group(2) ?? match.group(3) ?? '';
      final String url = match.group(2) ?? match.group(3) ?? '';

      spans.add(
        WidgetSpan(
          alignment: PlaceholderAlignment.baseline,
          baseline: TextBaseline.alphabetic,
          child: InkWell(
            onTap: () => _handleUrlClick(url),
            child: Text(
              label,
              style: const TextStyle(
                color: AppColors.secondary,
                fontWeight: FontWeight.w600,
                decoration: TextDecoration.underline,
              ),
            ),
          ),
        ),
      );

      lastIndex = match.end;
    }

    if (lastIndex < inputStr.length) {
      spans.add(TextSpan(text: inputStr.substring(lastIndex)));
    }

    return SelectableText.rich(
      TextSpan(
        children: spans,
        style: const TextStyle(
          fontSize: 14,
          height: 1.5,
          color: AppColors.onSurface,
        ),
      ),
    );
  }
}
