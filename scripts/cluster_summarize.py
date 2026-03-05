#!/usr/bin/env python3
"""
Smart document clustering and summarization without AI.
Uses TF-IDF + K-Means for clustering, LexRank for summaries.
"""

import os
import glob
import re
import json
from collections import Counter
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.cluster import KMeans
from sumy.summarizers.lex_rank import LexRankSummarizer
from sumy.parsers.plaintext import PlaintextParser
from sumy.nlp.tokenizers import Tokenizer
import nltk

# Download required NLTK data
nltk.download('punkt_tab', quiet=True)
nltk.download('stopwords', quiet=True)

INPUT_DIR = "exports/docs.openclaw.ai"
OUTPUT_DIR = "clustered_summaries"
N_CLUSTERS = 100
SUMMARY_SENTENCES = 8
KEY_POINTS_COUNT = 5
MIN_CONTENT_LENGTH = 200  # Minimum characters for a valid document


def clean_markdown(content):
    """Remove navigation links, footers, and other noise from markdown."""
    # Remove markdown links but keep text: [text](url) -> text
    content = re.sub(r'\[([^\]]+)\]\([^)]+\)', r'\1', content)

    # Remove horizontal rules
    content = re.sub(r'\n---+\n', '\n', content)

    # Remove multiple blank lines
    content = re.sub(r'\n{3,}', '\n\n', content)

    # Remove lines that are just links or navigation
    lines = content.split('\n')
    cleaned_lines = []
    skip_until_double_newline = False

    for line in lines:
        stripped = line.strip()

        # Skip empty lines but preserve paragraph breaks
        if not stripped:
            if cleaned_lines and cleaned_lines[-1]:
                cleaned_lines.append('')
            continue

        # Skip lines that are mostly links/navigation
        link_count = stripped.count('http')
        if link_count > 3:  # Line with many URLs is likely navigation
            continue

        # Skip lines that look like breadcrumb navigation
        if re.match(r'^[\[\/\]\s\-\>]+$', stripped):
            continue

        # Skip "404" error pages
        if stripped == '404' or 'couldn\'t find the page' in stripped.lower():
            continue

        # Skip "Was this page helpful?" type lines
        if 'helpful' in stripped.lower() and len(stripped) < 50:
            continue

        cleaned_lines.append(line)

    return '\n'.join(cleaned_lines).strip()


def extract_title_from_content(content, filename):
    """Extract title from markdown heading or filename."""
    lines = content.strip().split('\n')

    # Look for markdown heading
    for line in lines[:10]:
        line = line.strip()
        if line.startswith('# '):
            return line[2:].strip()
        if line.startswith('## '):
            return line[3:].strip()

    # Fallback to filename
    return filename.replace('.md', '').replace('-', ' ').title()


def load_documents():
    """Load all markdown documents from exports directory."""
    docs = []
    files = []
    titles = []

    for path in glob.glob(f"{INPUT_DIR}/*.md"):
        try:
            with open(path, 'r', encoding='utf-8') as f:
                raw_content = f.read()

            # Clean the content
            content = clean_markdown(raw_content)

            # Skip short documents (likely empty or 404 pages)
            if len(content) < MIN_CONTENT_LENGTH:
                continue

            # Skip documents that are mostly navigation
            text_only = re.sub(r'[\s\-\*\|]+', ' ', content)
            words = text_only.split()
            if len(words) < 50:  # Too few words
                continue

            docs.append(content)
            files.append(path)
            titles.append(extract_title_from_content(content, os.path.basename(path)))

        except Exception as e:
            print(f"Error loading {path}: {e}")
            continue

    return docs, files, titles


def extract_key_terms(content, top_n=10):
    """Extract key terms using frequency."""
    stopwords = set(nltk.corpus.stopwords.words('english'))

    # Tokenize
    words = re.findall(r'\b[a-zA-Z]{3,}\b', content.lower())
    words = [w for w in words if w not in stopwords]

    # Count frequencies
    word_counts = Counter(words)

    # Filter out common web/docs terms
    web_terms = {'http', 'https', 'www', 'com', 'org', 'docs', 'page', 'content',
                 'using', 'can', 'will', 'use', 'also', 'may', 'like', 'get',
                 'one', 'two', 'see', 'set', 'run', 'file', 'new', 'open'}
    for term in web_terms:
        if term in word_counts:
            del word_counts[term]

    return [word for word, _ in word_counts.most_common(top_n)]


def extract_key_points(content, n=5):
    """Extract key points as bullet-worthy sentences."""
    sentences = nltk.sent_tokenize(content)

    # Score sentences
    scored = []
    for i, sent in enumerate(sentences[:50]):
        sent = sent.strip()
        if len(sent) < 30 or len(sent) > 250:
            continue

        score = 0

        # Position bonus
        score += max(0, 10 - i)

        # Ideal length
        if 50 < len(sent) < 180:
            score += 5

        # Important indicators
        indicators = ['must', 'should', 'need', 'required', 'important',
                      'note', 'config', 'install', 'setup', 'create',
                      'enable', 'default', 'option', 'example', 'command']
        for ind in indicators:
            if ind in sent.lower():
                score += 2

        # Avoid questions
        if not sent.endswith('?'):
            score += 2

        scored.append((sent, score))

    scored.sort(key=lambda x: x[1], reverse=True)
    return [s[0] for s in scored[:n]]


def summarize_text(content, sentences=8):
    """Summarize text using LexRank."""
    try:
        parser = PlaintextParser.from_string(content, Tokenizer("english"))
        summarizer = LexRankSummarizer()
        summary = summarizer(parser.document, sentences)
        return " ".join([str(s) for s in summary])
    except:
        sents = nltk.sent_tokenize(content)
        return " ".join(sents[:sentences])


def detect_topic(key_terms, content):
    """Detect topic from key terms."""
    topic_indicators = {
        'install': 'Installation',
        'setup': 'Setup & Configuration',
        'config': 'Configuration',
        'api': 'API Reference',
        'guide': 'Guide',
        'cli': 'CLI Reference',
        'command': 'CLI Reference',
        'security': 'Security',
        'auth': 'Authentication',
        'database': 'Database',
        'deploy': 'Deployment',
        'docker': 'Docker',
        'channel': 'Channels',
        'webhook': 'Webhooks',
        'plugin': 'Plugins',
        'agent': 'Agent',
        'tool': 'Tools',
        'session': 'Session',
        'gateway': 'Gateway',
        'memory': 'Memory',
        'browser': 'Browser',
        'message': 'Messaging',
    }

    for term in key_terms[:5]:
        for indicator, topic in topic_indicators.items():
            if indicator in term:
                return topic

    if key_terms:
        return key_terms[0].title()

    return "Documentation"


def cluster_documents(docs, n_clusters=100):
    """Cluster documents using TF-IDF + K-Means."""
    print(f"Vectorizing {len(docs)} documents...")

    vectorizer = TfidfVectorizer(
        max_features=5000,
        stop_words='english',
        ngram_range=(1, 2),
        min_df=2,
        max_df=0.9
    )
    X = vectorizer.fit_transform(docs)

    print(f"Clustering into {n_clusters} groups...")
    kmeans = KMeans(
        n_clusters=n_clusters,
        random_state=42,
        n_init=10,
        max_iter=300
    )
    labels = kmeans.fit_predict(X)

    return labels, vectorizer


def process_cluster(cluster_id, items):
    """Process a single cluster and generate smart summary."""
    merged_content = "\n\n---\n\n".join([doc for _, doc, _ in items])
    first_title = items[0][2]

    key_terms = extract_key_terms(merged_content, top_n=10)
    topic = detect_topic(key_terms, merged_content)
    key_points = extract_key_points(merged_content, KEY_POINTS_COUNT)
    summary = summarize_text(merged_content, SUMMARY_SENTENCES)

    sources = [os.path.basename(f).replace('.md', '') for f, _, _ in items]

    return {
        'cluster_id': cluster_id,
        'title': first_title,
        'topic': topic,
        'key_terms': key_terms,
        'key_points': key_points,
        'summary': summary,
        'sources': sources,
        'doc_count': len(items)
    }


def format_output(data):
    """Format cluster data into readable output."""
    lines = []

    lines.append(f"=== {data['title']} ===")
    lines.append("")
    lines.append(f"TOPIC: {data['topic']}")
    lines.append(f"DOCUMENTS: {data['doc_count']}")
    lines.append("")

    lines.append("KEY POINTS:")
    for point in data['key_points']:
        point = point.strip().replace('\n', ' ')
        if len(point) > 200:
            point = point[:197] + "..."
        lines.append(f"• {point}")
    lines.append("")

    lines.append("SUMMARY:")
    summary_text = data['summary']
    sentences = nltk.sent_tokenize(summary_text)
    paragraph = ""
    for sent in sentences:
        if len(paragraph) + len(sent) > 400:
            lines.append(paragraph.strip())
            lines.append("")
            paragraph = sent + " "
        else:
            paragraph += sent + " "
    if paragraph:
        lines.append(paragraph.strip())
    lines.append("")

    lines.append(f"KEY TERMS: {', '.join(data['key_terms'])}")
    lines.append("")

    lines.append("SOURCES:")
    for source in data['sources'][:10]:
        lines.append(f"  - {source}")
    if len(data['sources']) > 10:
        lines.append(f"  ... and {len(data['sources']) - 10} more")

    return "\n".join(lines)


def main():
    os.makedirs(OUTPUT_DIR, exist_ok=True)

    print("Loading documents...")
    docs, files, titles = load_documents()
    print(f"Loaded {len(docs)} documents")

    if len(docs) == 0:
        print("No documents found!")
        return

    labels, vectorizer = cluster_documents(docs, N_CLUSTERS)

    clusters = {}
    for i, label in enumerate(labels):
        if label not in clusters:
            clusters[label] = []
        clusters[label].append((files[i], docs[i], titles[i]))

    print(f"Created {len(clusters)} clusters")

    print("Generating smart summaries...")
    for cluster_id, items in clusters.items():
        data = process_cluster(cluster_id, items)
        output = format_output(data)

        output_path = f"{OUTPUT_DIR}/cluster_{cluster_id:03d}.txt"
        with open(output_path, 'w', encoding='utf-8') as f:
            f.write(output)

        print(f"  Cluster {cluster_id}: {data['doc_count']} docs -> {data['topic']}")

    print(f"\nDone! Created {len(clusters)} files in {OUTPUT_DIR}/")


if __name__ == "__main__":
    main()
