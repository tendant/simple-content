#!/bin/bash
set -e

COMMAND="${1:-help}"

case "$COMMAND" in
  start)
    echo "Starting development services (Postgres + MinIO)..."
    docker-compose up -d postgres minio
    echo ""
    echo "Waiting for services to be healthy..."
    sleep 5
    echo ""
    echo "Services started successfully!"
    echo "  - Postgres: localhost:5433 (user: content, password: contentpass, db: simple_content)"
    echo "  - MinIO: localhost:9000 (console: localhost:9001, credentials: minioadmin/minioadmin)"
    echo ""
    echo "Next steps:"
    echo "  1. Run migrations: ./scripts/run-migrations.sh up"
    echo "  2. Create S3 bucket: aws --endpoint-url http://localhost:9000 s3 mb s3://content-bucket"
    echo "  3. Run tests: DATABASE_TYPE=postgres DATABASE_URL='postgresql://content:contentpass@localhost:5433/simple_content?sslmode=disable&search_path=content' go test ./pkg/simplecontent/..."
    ;;

  stop)
    echo "Stopping development services..."
    docker-compose down
    echo "Services stopped."
    ;;

  restart)
    echo "Restarting development services..."
    docker-compose restart postgres minio
    echo "Services restarted."
    ;;

  logs)
    docker-compose logs -f postgres minio
    ;;

  clean)
    echo "WARNING: This will remove all data volumes!"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
      docker-compose down -v
      echo "All services stopped and data removed."
    else
      echo "Cancelled."
    fi
    ;;

  status)
    echo "Service status:"
    docker-compose ps postgres minio
    ;;

  *)
    echo "Simple Content Development Environment"
    echo ""
    echo "Usage: $0 {start|stop|restart|logs|clean|status}"
    echo ""
    echo "Commands:"
    echo "  start   - Start Postgres and MinIO services"
    echo "  stop    - Stop all services"
    echo "  restart - Restart services"
    echo "  logs    - Follow service logs"
    echo "  clean   - Stop services and remove data volumes"
    echo "  status  - Show service status"
    echo ""
    exit 1
    ;;
esac
