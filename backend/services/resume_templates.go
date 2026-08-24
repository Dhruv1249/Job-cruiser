package services

import (
	"context"
	"fmt"
	"path"
	"strings"
)

const defaultResumeTemplatePath = "templates/resume.tex"
const defaultCoverLetterTemplatePath = "templates/cover_letter.tex"

/*
GetDefaultResumeTemplate returns an industry-standard, ATS-compliant, single-page LaTeX resume baseline template with custom macros.
*/
func GetDefaultResumeTemplate() string {
	return `\documentclass[letterpaper,10pt]{article}

\usepackage{latexsym}
\usepackage[empty]{fullpage}
\usepackage{titlesec}
\usepackage{marvosym}
\usepackage[usenames,dvipsnames]{color}
\usepackage{verbatim}
\usepackage{enumitem}
\usepackage[hidelinks]{hyperref}
\usepackage{fancyhdr}
\usepackage[english]{babel}
\usepackage{tabularx}
\usepackage{geometry}
\geometry{left=0.5in,top=0.45in,right=0.5in,bottom=0.45in}

\pagestyle{fancy}
\fancyhf{}
\fancyfoot{}
\renewcommand{\headrulewidth}{0pt}
\renewcommand{\footrulewidth}{0pt}

\urlstyle{same}

\raggedbottom
\raggedright
\setlength{\tabcolsep}{0in}

\titleformat{\section}{
  \vspace{-6pt}\scshape\raggedright\large
}{}{0em}{}[\color{black}\titlerule \vspace{-4pt}]

\newcommand{\resumeItem}[1]{
  \item\small{
    {#1 \vspace{-2pt}}
  }
}

\newcommand{\resumeSubheading}[4]{
  \vspace{-2pt}\item
    \begin{tabular*}{0.97\textwidth}[t]{l@{\extracolsep{\fill}}r}
      \textbf{#1} & #2 \\
      \textit{\small#3} & \textit{\small #4} \\
    \end{tabular*}\vspace{-5pt}
}

\newcommand{\resumeProjectHeading}[2]{
    \vspace{-2pt}\item
    \begin{tabular*}{0.97\textwidth}{l@{\extracolsep{\fill}}r}
      \small#1 & #2 \\
    \end{tabular*}\vspace{-5pt}
}

\newcommand{\resumeSubHeadingListStart}{\begin{itemize}[leftmargin=0.0in, label={}]}
\newcommand{\resumeSubHeadingListEnd}{\end{itemize}}
\newcommand{\resumeItemListStart}{\begin{itemize}[leftmargin=0.15in]}
\newcommand{\resumeItemListEnd}{\end{itemize}\vspace{-5pt}}

\begin{document}

\begin{center}
    \textbf{\Huge \scshape Candidate Name} \\ \vspace{2pt}
    \small +1 (555) 000-0000 $|$ \href{mailto:candidate@example.com}{\underline{candidate@example.com}} $|$ 
    \href{https://linkedin.com/in/candidate}{\underline{linkedin.com/in/candidate}} $|$
    \href{https://github.com/candidate}{\underline{github.com/candidate}}
\end{center}

\section{Summary}
\small{Results-driven Software Engineer with extensive experience designing and deploying scalable backend services, distributed systems, and modern cloud infrastructure. Proven track record of architecting performant REST/gRPC APIs, optimizing database operations, and collaborating in agile product teams.}

\section{Technical Skills}
 \begin{itemize}[leftmargin=0.15in, label={}]
    \small{\item{
     \textbf{Languages}{: Go, Python, TypeScript, Java, SQL, C++} \\
     \textbf{Frameworks \& Libraries}{: Gin, FastApi, React, Node.js, Next.js, gRPC} \\
     \textbf{Cloud \& DevOps}{: Docker, Kubernetes, AWS, Google Cloud, Terraform, CI/CD, Linux} \\
     \textbf{Databases \& Storage}{: PostgreSQL, Redis, MongoDB, Elasticsearch}
    }}
 \end{itemize}

\section{Experience}
  \resumeSubHeadingListStart

    \resumeSubheading
      {Senior Software Engineer}{Jan 2024 -- Present}
      {Acme Cloud Technologies}{San Francisco, CA}
      \resumeItemListStart
        \resumeItem{Architected and delivered high-throughput microservices handling 50M+ daily events using Go and Kafka, reducing p99 latency by 35\%.}
        \resumeItem{Engineered distributed caching layer using Redis clusters, cutting database read load by 60\% during peak traffic periods.}
        \resumeItem{Automated multi-region container deployments with Kubernetes and GitHub Actions, achieving zero-downtime releases.}
      \resumeItemListEnd

    \resumeSubheading
      {Software Engineer}{Jul 2022 -- Dec 2023}
      {Nexus Systems}{Austin, TX}
      \resumeItemListStart
        \resumeItem{Developed scalable RESTful APIs in Go and PostgreSQL supporting core customer identity, billing, and access management services.}
        \resumeItem{Optimized SQL indexing strategies and complex transactional queries across 200GB+ Postgres datasets, improving query execution by 40\%.}
        \resumeItem{Led migration of monolithic legacy components into modular containerized services using Docker and gRPC.}
      \resumeItemListEnd

  \resumeSubHeadingListEnd

\section{Projects}
    \resumeSubHeadingListStart
      \resumeProjectHeading
          {\textbf{Distributed Task Queue Engine} $|$ \emph{Go, Redis, PostgreSQL, Docker}}{\href{https://github.com/candidate/task-queue}{\underline{GitHub}}}
          \resumeItemListStart
            \resumeItem{Built a resilient distributed asynchronous task scheduler in Go supporting priority queues, exponential backoff retries, and dead-letter queues.}
            \resumeItem{Integrated worker heartbeat telemetry and Prometheus metrics exporter for real-time observability across cluster workers.}
          \resumeItemListEnd
      \resumeProjectHeading
          {\textbf{Real-Time Log Aggregator} $|$ \emph{Go, WebSockets, Elasticsearch, React}}{\href{https://github.com/candidate/log-aggregator}{\underline{GitHub}}}
          \resumeItemListStart
            \resumeItem{Engineered a real-time streaming log processor capable of indexing 10,000 log records per second with live browser dashboard streaming.}
          \resumeItemListEnd
    \resumeSubHeadingListEnd

\section{Education}
  \resumeSubHeadingListStart
    \resumeSubheading
      {Bachelor of Science in Computer Science}{2018 -- 2022}
      {State University of Technology}{City, State}
  \resumeSubHeadingListEnd

\end{document}`
}

/*
GetDefaultCoverLetterTemplate returns a professional, elegant LaTeX cover letter baseline template.
*/
func GetDefaultCoverLetterTemplate() string {
	return `\documentclass[11pt,a4paper]{article}
\usepackage[utf8]{inputenc}
\usepackage[empty]{fullpage}
\usepackage{geometry}
\usepackage{hyperref}
\usepackage{xcolor}

\geometry{left=0.8in,top=0.8in,right=0.8in,bottom=0.8in}
\setlength{\parindent}{0pt}
\setlength{\parskip}{10pt}

\hypersetup{
    colorlinks=true,
    linkcolor=black,
    urlcolor=blue!70!black
}

\begin{document}

\begin{center}
    {\LARGE \textbf{Candidate Name}} \\ \vspace{4pt}
    \small Location $|$ candidate@example.com $|$ +1 (555) 000-0000 $|$ \href{https://linkedin.com/in/candidate}{linkedin.com/in/candidate} $|$ \href{https://github.com/candidate}{github.com/candidate}
\end{center}

\vspace{10pt}
\hrule
\vspace{10pt}

\textbf{Hiring Team} \\
Target Company \\
Engineering Department

\vspace{5pt}

Dear Hiring Team,

I am writing to express my enthusiastic interest in the Software Engineer role at Target Company. With a solid foundation in modern software engineering, distributed systems, and scalable cloud architectures, I am eager to contribute to your engineering team's mission and technical goals.

Throughout my career, I have focused on designing reliable, high-performance backend systems and APIs. In my previous roles, I spearheaded key architectural initiatives, including reducing service response times by 35\% and optimizing data pipelines processing millions of daily transactions. My technical toolkit centers around Go, Python, PostgreSQL, Docker, and Kubernetes, enabling me to build maintainable, resilient services from day one.

Target Company's commitment to high engineering standards and technological innovation deeply resonates with me. I am particularly excited about the prospect of bringing my experience in backend development, database optimization, and cloud operations to support your product roadmap and scale your infrastructure.

Thank you for your time and consideration. I welcome the opportunity to discuss how my technical expertise and passion for engineering excellence can benefit Target Company.

\vspace{10pt}

Sincerely, \\
\vspace{15pt}
\textbf{Candidate Name}

\end{document}`
}

/*
EnsureDefaultTemplatesExist checks whether baseline LaTeX templates exist in the user's Open-Overleaf project and seeds them if missing.
*/
func EnsureDefaultTemplatesExist(ctx context.Context, mcpClient *MCPClient, projectName string) error {
	effectiveProject := strings.TrimSpace(projectName)
	if effectiveProject == "" {
		effectiveProject = "job_applications"
	}

	_, resumeReadError := mcpClient.ReadProjectFile(ctx, effectiveProject, defaultResumeTemplatePath)
	if resumeReadError != nil {
		writeError := mcpClient.WriteProjectFile(ctx, effectiveProject, defaultResumeTemplatePath, GetDefaultResumeTemplate())
		if writeError != nil {
			return fmt.Errorf("failed creating default resume template: %w", writeError)
		}
	}

	_, coverReadError := mcpClient.ReadProjectFile(ctx, effectiveProject, defaultCoverLetterTemplatePath)
	if coverReadError != nil {
		writeError := mcpClient.WriteProjectFile(ctx, effectiveProject, defaultCoverLetterTemplatePath, GetDefaultCoverLetterTemplate())
		if writeError != nil {
			return fmt.Errorf("failed creating default cover letter template: %w", writeError)
		}
	}

	return nil
}

/*
SyncAuxiliaryTemplateFiles discovers and copies non-tex template dependencies like .cls, .sty, and font assets into the target job folder.
*/
func SyncAuxiliaryTemplateFiles(ctx context.Context, mcpClient *MCPClient, projectName string, templateDir string, targetFolderPath string) error {
	effectiveProject := strings.TrimSpace(projectName)
	if effectiveProject == "" {
		effectiveProject = "job_applications"
	}

	cleanedTemplateDir := strings.Trim(templateDir, "/")
	if cleanedTemplateDir == "" {
		cleanedTemplateDir = "templates"
	}

	filesList, listError := mcpClient.ListFiles(ctx, effectiveProject, cleanedTemplateDir)
	if listError != nil {
		return nil
	}

	for _, fileEntry := range filesList {
		if fileEntry.IsDirectory {
			continue
		}
		filename := fileEntry.Name
		if strings.HasSuffix(strings.ToLower(filename), ".tex") {
			continue
		}

		sourceFilePath := path.Join(cleanedTemplateDir, filename)
		destinationFilePath := path.Join(targetFolderPath, filename)

		fileContent, readError := mcpClient.ReadProjectFile(ctx, effectiveProject, sourceFilePath)
		if readError != nil {
			continue
		}

		_ = mcpClient.WriteProjectFile(ctx, effectiveProject, destinationFilePath, fileContent)
	}

	return nil
}
