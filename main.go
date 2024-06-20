package main

import (
	"context"
	"fmt"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	ctx := context.Background()

	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		panic(err)
	}

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	}
	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)),
	)

	tracer := otel.Tracer("pgx-otel")
	ctx, span := tracer.Start(ctx, "pgx-otel")
	defer span.End()

	url := "postgres://usr:pw@localhost:5432/db?sslmode=disable"
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		panic(err)
	}

	cfg.ConnConfig.Tracer = otelpgx.NewTracer(otelpgx.WithTracerProvider(tp))

	conn, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	if err := conn.Ping(ctx); err != nil {
		panic(err)
	}

	row, err := conn.Query(ctx, "SELECT 1")
	if err != nil {
		panic(err)
	}
	defer row.Close()

	fmt.Println(row.RawValues())

	if err := tp.Shutdown(ctx); err != nil {
		panic(err)
	}
}
