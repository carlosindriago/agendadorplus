AgendadorPlus: Plan Integral de Proyecto

Sistema de Agendamiento Profesional
Community Edition (Open Source) + Enterprise Edition (SaaS)

1. RESUMEN EJECUTIVO

AgendadorPlus es un sistema de agendamiento profesional de doble naturaleza: una Community Edition (CE) de código abierto orientada al profesional independiente y una Enterprise Edition (EE) comercializada como SaaS multi-tenant para equipos y negocios. El proyecto se construye sobre un stack moderno (Go + React), adoptando Arquitectura Hexagonal y TDD.

La Regla de Oro: La CE debe ser perfecta (y gratuita) para un usuario individual. La EE es la evolución natural y obligatoria para equipos y negocios que requieran pagos y colaboración.

2. AUDITORÍA SENIOR DE FOCO (AJUSTE MVP)

Para garantizar el éxito en 3 semanas, aplicamos un filtro estricto de priorización:

🟢 [MANTENER] Arquitectura Hexagonal y TDD en el Core: Es nuestra vitrina técnica. No se negocia.

🟢 [MANTENER] Bloqueo Transaccional (Row-Level Lock): Es el corazón del sistema para evitar el double-booking.

🟡 [CAMBIAR] "Worker Pools" Complejos por Goroutines Simples con Recover: Usaremos una simple goroutine segura (defer recover()) y un mock de email (loguear en consola). Nota técnica: No habrá reintentos automáticos en caso de fallo para el MVP.

🟡 [CAMBIAR] Foco hacia Zonas Horarias y UX: El backend habla 100% en UTC. El frontend envía explícitamente su zona horaria (ej. America/Lima) y se encarga de la conversión visual usando date-fns-tz y react-day-picker.

🔴 [ELIMINAR] Autenticación Compleja en CE: Un solo administrador con Basic Auth o JWT simple estático.

🚫 FUERA DEL SCOPE (NON-MVP) - El Manifiesto Anti-Scope Creep:
Para garantizar la entrega, queda estrictamente prohibido incluir en las primeras 3 semanas:

Roles avanzados y permisos granulares.

Branding complejo o temas dinámicos.

Multi-account (Equipos).

Pasarelas de pago.

Modelo de agendamiento por "rangos dinámicos" con buffers complejos (usaremos un modelo temporal de time_slots pre-calculados por simplicidad).

3. PLANTEAMIENTO DEL PROBLEMA Y OBJETIVOS

Problema: Herramientas actuales son costosas, sufren de double-booking en implementaciones baratas, y el usuario sufre confusiones con las zonas horarias.

Objetivo de Ingeniería: Eliminar el double-booking mediante SELECT ... FOR UPDATE NOWAIT en PostgreSQL, manteniendo la transacción mínima y rápida.

Objetivo de Negocio: Alcanzar un MVP desplegable y usable por un humano real en 3 Sprints de 1 semana ("Modo Francotirador").

4. ARQUITECTURA TÉCNICA Y DISEÑO

4.1 El Stack Elegido

Backend: Go (Golang) + Gin Framework + PostgreSQL (Neon DB).

Frontend: React 18 + Vite + TanStack Query + TailwindCSS + shadcn/ui.

Infraestructura (CE): Docker + Docker Compose + GitHub Actions (CI/CD).

4.2 Diseño de Sistema (System Sequence Diagram - SSD)

Flujo Crítico de Reserva Segura (Nivel Senior - Optimizado):

Cliente UI solicita horarios. Envía explícitamente el header X-Timezone: America/Lima.

Backend calcula slots disponibles en UTC y los retorna. Frontend renderiza las horas locales.

Cliente elige 15:00 hrs (Local) -> Frontend convierte a UTC y envía POST /api/v1/appointments.

Un Middleware de Gin (gin-contrib/timeout) envuelve la petición garantizando un timeout estricto de 3 segundos sin repetir código.

[ZONA CRÍTICA] Caso de Uso abre Transacción. Repositorio ejecuta transaccion mínima: SELECT * FROM time_slots WHERE id=? FOR UPDATE NOWAIT.

Si está libre, inserta la reserva, hace commit rápido y libera. Si no, rollback y HTTP 409 con error estructurado {"error": "slot_locked"}.

Goroutine de notificación (simulada) se lanza en background.

Handler responde HTTP 201 Created.

5. METODOLOGÍA: TDD Y CLEAN CODE

El TDD incluirá pruebas de integración críticas ("Testcontainers"). Todas se ejecutarán con el flag -race de Go:

Test Concurrencia: 5 goroutines atacan el mismo slot exacto. 1 éxito (201), 4 fallos (409).

Test Timeout / Abort: Forzar latencia > 3s en el mock para verificar que el context expira y retorna 504/408 rápidamente, liberando recursos.

Test Overlapping: Verificar que elegir slots adyacentes (donde el fin de uno coincide con el inicio del otro en el mismo milisegundo) no rompa la búsqueda de disponibilidad.

Test Zona Horaria: Validar que un slot reservado desde Lima a las 15:00 se guarda correctamente en UTC y se recupera igual para un cliente en Madrid.

6. ROADMAP: LAS ETAPAS DEL MVP (3 SPRINTS)

Sprint 1: El Motor Central, DB y Zonas Horarias (Backend CE)

Día 1: Definición de BD. Migraciones con golang-migrate (001_init.sql). Incluir columna tenant_id y creación de índices optimizados.

Día 2-3: Foco total en TDD para la lógica de Zonas Horarias y disponibilidad. Todo slot se guarda en UTC.

Día 4: Adaptadores Postgres y API REST. Implementación de Gin Timeout Middleware y observabilidad mínima (Logging estructurado con slog).

Día 5: Mock (Logger) de notificaciones asíncronas con recover().

Sprint 2: La Experiencia de Usuario de Calendario (Frontend CE)

Día 1-2: Setup de Vite, React y shadcn/ui. Panel Admin ultraligero.

Día 3-4: Vista Pública de Reserva usando react-day-picker. Crucial: Mostrar la zona horaria detectada.

Día 5: Integración. Idempotencia UI: deshabilitar botón al primer clic y renderizar mensaje amigable y estructurado según el 409 (ej. "slot_taken" vs "slot_instantly_locked").

Sprint 3: Empaquetado, Confiabilidad y OSS

Día 1-2: Configuración de docker-compose.yml.

Día 3: Setup de CI/CD básico en GitHub Actions (correr golangci-lint, go test -race y migraciones en cada push).

Día 4: README.md documentando decisiones de arquitectura y deuda técnica asumida.

Día 5: Deploy gratuito (Vercel + Koyeb + NeonDB).

7. ESTRATEGIAS DE DESPLIEGUE (DEPLOYMENT)

7.1 Despliegue Gratuito (MVP / Portafolio)

Frontend: Vercel (maneja zonas horarias del cliente impecablemente).

Backend: Koyeb (sin cold-starts agresivos).

Base de Datos: Neon DB (PostgreSQL Serverless).

8. OBSERVACIONES TÉCNICAS (CONFIABILIDAD OPERACIONAL)

Zonas Horarias (La trampa #1): El Backend solo entiende UTC. Se exige explícitamente el header de zona horaria al frontend.

Deuda Técnica Documentada (Modelo de Datos): Para el MVP, usaremos time_slots pre-calculados. Esto limita duraciones variables o buffers dinámicos. En la versión EE, la tabla time_slots será reemplazada o extendida hacia un modelo de "búsqueda de superposición de rangos temporales reales".

9. FLUJO DE TRABAJO PROFESIONAL (GIT & CI/CD)

Aplicaremos GitHub Flow combinado con Conventional Commits. Esto demuestra madurez y permite automatizar la generación de Changelogs en el futuro.

Rama Principal: main (Siempre desplegable y protegida, no se hacen commits directos. Estrategia 'Squash and Merge' obligatoria en GitHub).

Ramas de Trabajo: feat/nombre-feature, fix/nombre-bug, chore/tarea-mantenimiento.

Convención de Commits: feat(api): ..., fix(db): ..., test(domain): ...

Flujo CI/CD MVP: Creas rama -> Commits convencionales -> PR -> Actions corre go test -race -> Merge a main -> Deploy automático.

10. ESQUEMA DE BASE DE DATOS NORMALIZADO (MVP)

Usaremos PostgreSQL. Todos los IDs serán UUIDv4 para evitar enumeración y facilitar la replicación futura (EE).

-- Tabla base para el futuro Multi-Tenant (En MVP, existirá 1 solo registro por defecto)
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Profesionales/Dueños de la agenda
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL, -- Para MVP auth básica
    name VARCHAR(255) NOT NULL
);

-- Servicios ofrecidos
CREATE TABLE services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    duration_mins INT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE
);

-- Bloques de tiempo (La clave de nuestro MVP pre-calculado)
CREATE TABLE time_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id), -- Añadido explícitamente al slot para búsquedas rápidas EE
    user_id UUID REFERENCES users(id),
    start_time_utc TIMESTAMPTZ NOT NULL,
    end_time_utc TIMESTAMPTZ NOT NULL CHECK (end_time_utc > start_time_utc),
    is_available BOOLEAN DEFAULT TRUE,
    UNIQUE(user_id, start_time_utc) -- Evita solapamiento físico en BD de ranuras idénticas
);
-- Índice optimizado para consultas de disponibilidad por clínica/tenant
CREATE INDEX idx_time_slots_tenant_date ON time_slots(tenant_id, start_time_utc);

-- Las reservas reales
CREATE TABLE appointments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- DEUDA TÉCNICA (MVP): 1 slot = 1 cita. Garantiza Row-Level Lock fácil.
    -- En el futuro EE, si se requiere multiplicity (ej. clases grupales), se removerá este UNIQUE.
    time_slot_id UUID REFERENCES time_slots(id) UNIQUE, 
    service_id UUID REFERENCES services(id),
    client_name VARCHAR(255) NOT NULL,
    client_email VARCHAR(255) NOT NULL,
    client_timezone VARCHAR(50) NOT NULL, -- Ej: 'America/Lima', vital para notificaciones
    status VARCHAR(50) DEFAULT 'confirmed',
    created_at TIMESTAMPTZ DEFAULT NOW()
);


11. FLUJOS DE USUARIO (USER JOURNEYS) Y PANTALLAS (SCOPE MVP)

Para evitar programar pantallas innecesarias, este es el límite visual del proyecto:

Flujo A: Profesional (Administrador) - 3 Pantallas

Login: Pantalla simple (Email / Pass).

Dashboard (Lista de Citas): Tabla sencilla (Fecha, Cliente, Servicio).

Gestión de Disponibilidad (Generador): Formulario donde elige "Lunes a Viernes de 9am a 5pm, slots de 30 mins" -> Botón "Generar".

Flujo B: Cliente Final (El que reserva) - 1 Pantalla (SPA)

Vista Pública (/book/:user_id): * Paso 1: Muestra calendario mensual (react-day-picker).

Paso 2: Muestra horas disponibles en su zona horaria (detectada automáticamente).

Paso 3: Formulario modal (Nombre, Email, Servicio). Botón "Confirmar" (Se deshabilita tras clic).

Paso 4: Pantalla/Toast de éxito integral ("Tu cita el 12 de Abril a las 15:00 (Hora Local) para el servicio X está confirmada a nombre de Y").

12. ESTRUCTURA DEL README.md INICIAL (PARA GITHUB Y PORTAFOLIO)

Título y Badges: (Go version, React version, License, Build Status).

El Problema (TL;DR): Falla en concurrencia (double-booking) en apps de agenda.

La Solución (AgendadorPlus): Arquitectura Hexagonal, Clean Code y Bloqueo Transaccional en Postgres.

Quickstart (Docker): git clone, copiar .env.example, docker compose up -d.

Arquitectura y Deuda Técnica: Diagrama de capas y justificación explícita del uso de slots fijos vs rangos para el MVP.

Manejo de Zonas Horarias: BD en UTC y conversión cliente local.

13. ANEXO: CÓDIGO DE REFERENCIA TDD (FASE RED)

Test principal para garantizar que no existan sobreventas (double-booking). Incluye sub-tests (t.Run) para mayor claridad y un test de expiración de Timeout.

package services_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// MockRepository simula nuestro Postgres con Row-Level Lock
type MockRepository struct {
	mu           sync.Mutex 
	slotOccupied bool
}

func (m *MockRepository) BookSlotWithinTransaction(ctx context.Context, slotID string) error {
	m.mu.Lock()         
	defer m.mu.Unlock() 

	// Simulación de latencia pesada si el test lo requiere (para forzar timeout)
	select {
	case <-ctx.Done():
		return ctx.Err() // Retorna context.DeadlineExceeded
	case <-time.After(10 * time.Millisecond):
		// Continuar ejecución normal
	}

	if m.slotOccupied {
		return errors.New("domain_error: time_slot_locked_or_unavailable")
	}

	m.slotOccupied = true
	return nil
}

func Test_BookAppointment_Concurrency(t *testing.T) {
	t.Run("SLOT_DISPONIBLE_UNO_EXITO_RESTO_FALLA", func(t *testing.T) {
		repo := &MockRepository{slotOccupied: false}
		var wg sync.WaitGroup
		var successCount int32
		var errorCount int32
		concurrentUsers := 10
		slotID := "uuid-viernes-1500" 

		for i := 0; i < concurrentUsers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()

				err := repo.BookSlotWithinTransaction(ctx, slotID)
				if err == nil {
					atomic.AddInt32(&successCount, 1)
				} else {
					atomic.AddInt32(&errorCount, 1)
				}
			}()
		}
		wg.Wait()

		if successCount != 1 {
			t.Errorf("Esperábamos 1 éxito, obtuvimos %d", successCount)
		}
		if expectedErrors := int32(concurrentUsers - 1); errorCount != expectedErrors {
			t.Errorf("Esperábamos %d errores, obtuvimos %d", expectedErrors, errorCount)
		}
	})

	t.Run("TIMEOUT_EXPIRADO_LIBERA_RECURSO", func(t *testing.T) {
		repo := &MockRepository{slotOccupied: false}
		
		// Contexto ridículamente corto para forzar el timeout antes de que el DB responda
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		err := repo.BookSlotWithinTransaction(ctx, "uuid-timeout-test")

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Esperábamos error por DeadlineExceeded, obtuvimos: %v", err)
		}
		if repo.slotOccupied {
			t.Errorf("El slot no debió marcarse como ocupado si la transacción fue cancelada por timeout")
		}
	})
}
