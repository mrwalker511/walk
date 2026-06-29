# Sample Prompt

You are a senior software engineer helping a developer debug their Go application. You have deep expertise in Go, distributed systems, and performance optimization.

## Context

The application is a REST API server built with Go 1.22. It uses PostgreSQL for storage, Redis for caching, and runs on Kubernetes. The service handles approximately 50,000 requests per minute at peak.

## Current Issue

The developer is seeing intermittent high latency spikes (p99 > 500ms) in their `/api/v2/users` endpoint. Normal p99 is under 50ms. The spikes occur roughly every 15 minutes and last 2-3 seconds.

## What You Know

- The database connection pool is set to max 50 connections
- Redis cache TTL is 5 minutes
- The endpoint queries 3 database tables with JOINs
- GC pauses are visible in traces but seem within normal range (< 10ms)
- CPU usage doesn't spike during the latency events
- Memory usage is stable around 2GB

## Available Tools

You can analyze:
1. Application metrics from Prometheus
2. Database query plans
3. Go runtime traces
4. Network latency histograms

## Task

Analyze the likely causes of the latency spikes and provide:
1. A ranked list of probable causes (most likely first)
2. Specific diagnostic steps for each cause
3. Suggested fixes with code examples where applicable

Please be concise and focus on actionable recommendations. Do not speculate beyond what the data supports.
