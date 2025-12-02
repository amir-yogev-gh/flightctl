# Flight Control - Technical Design Document

> **Document Type:** Technical Design  
> **Audience:** Engineers, Architects, Contributors  
> **Last Updated:** November 29, 2025

## Table of Contents

- [Overview](#overview)
- [Agent Implementation](#agent-implementation)
- [API Server Implementation](#api-server-implementation)
- [Worker System](#worker-system)
- [Data Store Layer](#data-store-layer)
- [Queue System](#queue-system)
- [Rollout Management](#rollout-management)
- [Resource Rendering](#resource-rendering)
- [Certificate Management](#certificate-management)
- [Observability](#observability)
- [Performance Considerations](#performance-considerations)

---

## Overview

This document provides in-depth technical details about Flight Control's implementation, focusing on core subsystems, algorithms, and design decisions.

---

## Agent Implementation

### Agent Lifecycle

The agent runs as a systemd service on each device with the following lifecycle:

```go
func (a *Agent) Run(ctx context.Context) error {
    // 1. Initialize TPM and identity
    tpmClient := loadTPM()
    identityProvider := identity.NewProvider(tpmClient, config)
    identityProvider.Initialize(ctx)
    
    // 2. Enroll or use existing identity
    if !enrolled {
        enrollment.Enroll(ctx, identityProvider)
    }
    
    // 3. Initialize managers
    specManager := spec.NewManager(client, store)
    statusManager := status.NewManager(client)
    
    // 4. Start controllers
    go osController.Run(ctx)
    go configController.Run(ctx)
    go appController.Run(ctx)
    go resourceMonitor.Run(ctx)
    go consoleManager.Run(ctx)
    
    // 5. Main reconciliation loop
    for {
        select {
        case <-ticker.C:
            reconcile(ctx, specManager, statusManager)
        case <-ctx.Done():
            return shutdown()
        }
    }
}
```

### Spec Management

**Desired State Handling**:

The agent maintains a cache of the desired device specification and tracks versions:

```go
type Manager struct {
    current  *v1beta1.DeviceSpec  // Current cached spec
    version  string                // Resource version
    store    SpecStore             // Persistent storage
    client   client.Client         // API client
}

func (m *Manager) Sync(ctx context.Context) error {
    // Fetch from server
    rendered, err := m.client.GetRenderedDeviceSpec(ctx, deviceName)
    
    // Check if changed
    if rendered.ResourceVersion == m.version {
        return nil // No changes
    }
    
    // Validate spec
    if err := validate(rendered.RenderedDeviceSpec); err != nil {
        return err
    }
    
    // Store locally
    m.store.Write(rendered)
    m.current = rendered.RenderedDeviceSpec
    m.version = rendered.ResourceVersion
    
    return nil
}
```

### OS Update Controller

**bootc Integration**:

```go
type Controller struct {
    executer executer.Executer
    status   *status.StatusManager
}

func (c *Controller) Reconcile(ctx context.Context, desired *OSSpec) error {
    // 1. Check current OS image
    current, err := c.getCurrentImage()
    if current == desired.Image {
        return nil
    }
    
    // 2. Switch to new image
    log.Infof("Switching to image: %s", desired.Image)
    cmd := exec.Command("bootc", "switch", "--retain", desired.Image)
    if err := cmd.Run(); err != nil {
        c.status.SetCondition(v1beta1.Condition{
            Type: "OSUpdateFailed",
            Status: "True",
            Message: err.Error(),
        })
        return err
    }
    
    // 3. Schedule reboot
    c.scheduleReboot()
    
    return nil
}
```

**Rollback Logic**:

The agent integrates with greenboot for automatic rollback:

```bash
# /etc/greenboot/check/required.d/01-flightctl-health.sh
#!/bin/bash
# Check if flightctl-agent is healthy
systemctl is-active flightctl-agent.service || exit 1

# Check custom health endpoints
/usr/local/bin/flightctl-health-check || exit 1

exit 0
```

### Application Management

**Container Support**:

The agent manages three types of container workloads:

1. **Standalone Containers** (Podman)
```go
func (c *AppController) deployContainer(app *v1beta1.Application) error {
    args := []string{"run", "-d", "--name", app.Name}
    
    // Add environment variables
    for k, v := range app.EnvVars {
        args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
    }
    
    // Add volumes
    for _, vol := range app.Volumes {
        args = append(args, "-v", fmt.Sprintf("%s:%s", vol.Source, vol.Target))
    }
    
    args = append(args, app.Image)
    
    return c.executer.Execute("podman", args...)
}
```

2. **Compose Applications** (docker-compose)
```go
func (c *AppController) deployCompose(app *v1beta1.Application) error {
    // Write compose file
    composePath := filepath.Join("/etc/flightctl/compose", app.Name, "compose.yaml")
    if err := os.WriteFile(composePath, []byte(app.ComposeContent), 0644); err != nil {
        return err
    }
    
    // Deploy with podman-compose
    return c.executer.Execute("podman-compose", "-f", composePath, "up", "-d")
}
```

3. **Quadlets** (systemd)
```go
func (c *AppController) deployQuadlet(app *v1beta1.Application) error {
    // Write quadlet file
    quadletPath := filepath.Join("/etc/containers/systemd", app.Name+".container")
    if err := os.WriteFile(quadletPath, []byte(app.QuadletContent), 0644); err != nil {
        return err
    }
    
    // Reload systemd
    if err := c.executer.Execute("systemctl", "daemon-reload"); err != nil {
        return err
    }
    
    // Start service
    return c.executer.Execute("systemctl", "start", app.Name+".service")
}
```

### Configuration Management

**File-Based Configuration**:

```go
type ConfigController struct {
    fileWriter fileio.Writer
    backup     BackupManager
}

func (c *ConfigController) Apply(configs []v1beta1.ConfigProviderSpec) error {
    for _, config := range configs {
        switch config.ConfigType {
        case "GitConfigProviderSpec":
            c.applyGitConfig(config.GitRef)
        case "InlineConfigProviderSpec":
            c.applyInlineConfig(config.Inline)
        case "KubernetesSecretProviderSpec":
            c.applyK8sSecret(config.SecretRef)
        }
    }
    return nil
}

func (c *ConfigController) applyInlineConfig(inline []v1beta1.FileSpec) error {
    for _, file := range inline {
        // Backup existing file
        if err := c.backup.BackupFile(file.Path); err != nil {
            return err
        }
        
        // Decode content
        content, err := base64.StdEncoding.DecodeString(file.Content)
        if err != nil {
            return err
        }
        
        // Write atomically
        if err := c.fileWriter.WriteFile(file.Path, content, os.FileMode(file.Mode)); err != nil {
            return err
        }
        
        // Set ownership
        if err := os.Chown(file.Path, file.User, file.Group); err != nil {
            return err
        }
    }
    return nil
}
```

### Status Reporting

**Heartbeat and Status Updates**:

```go
type StatusManager struct {
    client       client.Client
    deviceName   string
    lastStatus   *v1beta1.DeviceStatus
    ticker       *time.Ticker
}

func (m *StatusManager) Start(ctx context.Context) {
    m.ticker = time.NewTicker(30 * time.Second)
    
    for {
        select {
        case <-m.ticker.C:
            m.reportStatus(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (m *StatusManager) reportStatus(ctx context.Context) error {
    status := &v1beta1.DeviceStatus{
        LastSeen: time.Now(),
        SystemInfo: m.collectSystemInfo(),
        Resources: m.collectResources(),
        Applications: m.collectApplicationStatus(),
        Conditions: m.conditions,
    }
    
    // Only send if changed
    if reflect.DeepEqual(status, m.lastStatus) {
        return nil
    }
    
    err := m.client.UpdateDeviceStatus(ctx, m.deviceName, status)
    if err == nil {
        m.lastStatus = status
    }
    return err
}
```

### Lifecycle Hooks

**Hook Execution**:

```go
type HookManager struct {
    executer executer.Executer
}

func (h *HookManager) ExecuteHooks(hooks *v1beta1.LifecycleHooks, phase string) error {
    var hooksToRun []v1beta1.HookAction
    
    switch phase {
    case "before":
        hooksToRun = hooks.Before
    case "after":
        hooksToRun = hooks.After
    case "on-failure":
        hooksToRun = hooks.OnFailure
    }
    
    for _, hook := range hooksToRun {
        if err := h.executeHook(hook); err != nil {
            return fmt.Errorf("hook %s failed: %w", hook.Name, err)
        }
    }
    return nil
}

func (h *HookManager) executeHook(hook v1beta1.HookAction) error {
    switch hook.Type {
    case "command":
        return h.executer.ExecuteWithTimeout(hook.Command[0], hook.Command[1:]..., hook.Timeout)
    case "systemd":
        return h.executer.Execute("systemctl", hook.Action, hook.Unit)
    default:
        return fmt.Errorf("unknown hook type: %s", hook.Type)
    }
}
```

---

## API Server Implementation

### Request Flow

```
HTTP Request
    ↓
Chi Router
    ↓
oapi-codegen Validator (OpenAPI validation)
    ↓
Authentication Middleware (JWT or mTLS)
    ↓
Authorization Middleware (SpiceDB/RBAC)
    ↓
Rate Limiting Middleware
    ↓
Tracing Middleware (OpenTelemetry)
    ↓
Handler (service layer)
    ↓
Store Layer (database operations)
    ↓
Response
```

### Service Layer

**Device Service**:

```go
type DeviceService struct {
    store        store.Store
    queueProvider queues.Provider
    ca           *crypto.CA
}

func (s *DeviceService) UpdateDevice(ctx context.Context, orgID uuid.UUID, name string, device *v1beta1.Device) (*v1beta1.Device, error) {
    // 1. Validate device spec
    if err := device.Validate(); err != nil {
        return nil, flterrors.NewBadRequest("invalid device spec: %v", err)
    }
    
    // 2. Get existing device
    existing, err := s.store.Device().Get(ctx, orgID, name)
    if err != nil {
        return nil, err
    }
    
    // 3. Merge specs
    updated := existing.DeepCopy()
    updated.Spec = device.Spec
    updated.Metadata.Generation++
    
    // 4. Store in database
    if err := s.store.Device().Update(ctx, orgID, updated); err != nil {
        return nil, err
    }
    
    // 5. Enqueue rollout task (if needed)
    if specChanged(existing.Spec, updated.Spec) {
        task := &tasks.DeviceUpdateTask{
            DeviceName: name,
            OrgID: orgID,
        }
        s.queueProvider.Enqueue(ctx, "device-updates", task)
    }
    
    // 6. Record event
    s.recordEvent(ctx, orgID, "DeviceUpdated", name)
    
    return updated, nil
}
```

### Resource Rendering

**Template Rendering**:

The server renders device specs by merging fleet templates with device-specific configs:

```go
type Renderer struct {
    store store.Store
}

func (r *Renderer) RenderDevice(ctx context.Context, device *v1beta1.Device) (*v1beta1.RenderedDeviceSpec, error) {
    rendered := &v1beta1.RenderedDeviceSpec{
        Config:       []v1beta1.ConfigProviderSpec{},
        Applications: []v1beta1.ApplicationSpec{},
    }
    
    // 1. Get all fleets that match this device
    fleets, err := r.store.Fleet().GetBySelector(ctx, device.Metadata.Labels)
    if err != nil {
        return nil, err
    }
    
    // 2. Apply fleet templates in order (oldest to newest)
    sort.Slice(fleets, func(i, j int) bool {
        return fleets[i].Metadata.CreationTimestamp.Before(fleets[j].Metadata.CreationTimestamp)
    })
    
    for _, fleet := range fleets {
        r.mergeTemplate(rendered, fleet.Spec.Template.Spec)
    }
    
    // 3. Apply device-specific config (highest priority)
    if device.Spec != nil {
        r.mergeSpec(rendered, device.Spec)
    }
    
    // 4. Resolve config references
    if err := r.resolveConfigRefs(ctx, rendered); err != nil {
        return nil, err
    }
    
    return rendered, nil
}

func (r *Renderer) mergeSpec(target, source *v1beta1.DeviceSpec) {
    // OS image: device overrides fleet
    if source.Os != nil && source.Os.Image != "" {
        target.Os = source.Os
    }
    
    // Config: append (all configs are applied)
    target.Config = append(target.Config, source.Config...)
    
    // Applications: merge by name
    for _, app := range source.Applications {
        idx := findApp(target.Applications, app.Name)
        if idx >= 0 {
            target.Applications[idx] = app // Override
        } else {
            target.Applications = append(target.Applications, app) // Add
        }
    }
}
```

### WebSocket Support

**Real-Time Updates**:

```go
type WSHandler struct {
    connections sync.Map // deviceName -> *websocket.Conn
    pubsub      *redis.PubSub
}

func (h *WSHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    
    deviceName := chi.URLParam(r, "name")
    h.connections.Store(deviceName, conn)
    defer h.connections.Delete(deviceName)
    
    // Subscribe to device updates
    ch := h.pubsub.Subscribe(ctx, "device:"+deviceName)
    
    for msg := range ch {
        conn.WriteJSON(msg.Payload)
    }
}

func (h *WSHandler) NotifyDeviceUpdate(deviceName string, update interface{}) {
    if conn, ok := h.connections.Load(deviceName); ok {
        conn.(*websocket.Conn).WriteJSON(update)
    }
}
```

---

## Worker System

### Task Queue Architecture

**Redis Streams-Based Queue**:

```go
type RedisQueue struct {
    client *redis.Client
    stream string
}

func (q *RedisQueue) Enqueue(ctx context.Context, task Task) error {
    data, err := json.Marshal(task)
    if err != nil {
        return err
    }
    
    return q.client.XAdd(ctx, &redis.XAddArgs{
        Stream: q.stream,
        Values: map[string]interface{}{
            "type": task.Type(),
            "data": string(data),
            "enqueued_at": time.Now().Unix(),
        },
    }).Err()
}

func (q *RedisQueue) Consume(ctx context.Context, handler TaskHandler) error {
    // Create consumer group
    q.client.XGroupCreate(ctx, q.stream, "workers", "0")
    
    for {
        // Read from stream
        streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
            Group:    "workers",
            Consumer: workerID,
            Streams:  []string{q.stream, ">"},
            Count:    10,
            Block:    time.Second,
        }).Result()
        
        for _, stream := range streams {
            for _, msg := range stream.Messages {
                q.processMessage(ctx, msg, handler)
            }
        }
    }
}

func (q *RedisQueue) processMessage(ctx context.Context, msg redis.XMessage, handler TaskHandler) {
    defer func() {
        if r := recover(); r != nil {
            log.Errorf("Task panic: %v", r)
            q.moveToDeadLetter(msg)
        }
    }()
    
    // Process task
    if err := handler.Handle(ctx, msg.Values["data"].(string)); err != nil {
        log.Errorf("Task failed: %v", err)
        
        // Retry logic
        retries := msg.Values["retries"].(int)
        if retries < 3 {
            q.requeueWithBackoff(msg, retries+1)
        } else {
            q.moveToDeadLetter(msg)
        }
    } else {
        // Acknowledge success
        q.client.XAck(ctx, q.stream, "workers", msg.ID)
    }
}
```

### Worker Pool

```go
type WorkerPool struct {
    size     int
    queue    Queue
    handlers map[string]TaskHandler
}

func (p *WorkerPool) Start(ctx context.Context) {
    for i := 0; i < p.size; i++ {
        go p.worker(ctx, i)
    }
}

func (p *WorkerPool) worker(ctx context.Context, id int) {
    log.Infof("Worker %d started", id)
    defer log.Infof("Worker %d stopped", id)
    
    for {
        select {
        case task := <-p.queue.Dequeue(ctx):
            handler := p.handlers[task.Type()]
            if handler == nil {
                log.Errorf("No handler for task type: %s", task.Type())
                continue
            }
            
            if err := handler.Handle(ctx, task); err != nil {
                log.Errorf("Worker %d task failed: %v", id, err)
            }
        case <-ctx.Done():
            return
        }
    }
}
```

### Task Handlers

**Rollout Task Handler**:

```go
type RolloutHandler struct {
    store store.Store
}

func (h *RolloutHandler) Handle(ctx context.Context, task *RolloutTask) error {
    fleet, err := h.store.Fleet().Get(ctx, task.OrgID, task.FleetName)
    if err != nil {
        return err
    }
    
    // Get all devices in fleet
    devices, err := h.store.Device().GetByFleet(ctx, task.OrgID, task.FleetName)
    if err != nil {
        return err
    }
    
    // Calculate rollout strategy
    strategy := fleet.Spec.Policy.RolloutPolicy
    batches := h.calculateBatches(devices, strategy)
    
    // Execute rollout in batches
    for i, batch := range batches {
        log.Infof("Rolling out batch %d/%d (%d devices)", i+1, len(batches), len(batch))
        
        // Update devices
        for _, device := range batch {
            if err := h.updateDevice(ctx, device, fleet.Spec.Template); err != nil {
                log.Errorf("Failed to update device %s: %v", device.Metadata.Name, err)
            }
        }
        
        // Wait for batch to complete
        if err := h.waitForBatch(ctx, batch, fleet.Spec.Policy.ProgressDeadlineSeconds); err != nil {
            return fmt.Errorf("batch %d failed: %w", i+1, err)
        }
    }
    
    return nil
}

func (h *RolloutHandler) calculateBatches(devices []*v1beta1.Device, policy *v1beta1.RolloutPolicy) [][]*v1beta1.Device {
    total := len(devices)
    maxUnavailable := calculateMaxUnavailable(total, policy.MaxUnavailable)
    
    batches := [][]*v1beta1.Device{}
    for i := 0; i < total; i += maxUnavailable {
        end := min(i+maxUnavailable, total)
        batches = append(batches, devices[i:end])
    }
    
    return batches
}
```

---

## Data Store Layer

### Store Interface

```go
type Store interface {
    Device() DeviceStore
    Fleet() FleetStore
    Repository() RepositoryStore
    // ... other resources
}

type DeviceStore interface {
    Create(ctx context.Context, orgID uuid.UUID, device *v1beta1.Device) error
    Get(ctx context.Context, orgID uuid.UUID, name string) (*v1beta1.Device, error)
    Update(ctx context.Context, orgID uuid.UUID, device *v1beta1.Device) error
    Delete(ctx context.Context, orgID uuid.UUID, name string) error
    List(ctx context.Context, orgID uuid.UUID, opts ListOptions) ([]*v1beta1.Device, error)
    UpdateStatus(ctx context.Context, orgID uuid.UUID, name string, status *v1beta1.DeviceStatus) error
}
```

### GORM Implementation

```go
type DeviceStoreImpl struct {
    db  *gorm.DB
    log *logrus.Entry
}

func (s *DeviceStoreImpl) Get(ctx context.Context, orgID uuid.UUID, name string) (*v1beta1.Device, error) {
    ctx, span := tracing.StartSpan(ctx, "store.Device.Get")
    defer span.End()
    
    var model DeviceModel
    result := s.db.WithContext(ctx).
        Where("org_id = ? AND name = ?", orgID, name).
        First(&model)
    
    if result.Error != nil {
        if errors.Is(result.Error, gorm.ErrRecordNotFound) {
            return nil, flterrors.NewNotFound("device %s not found", name)
        }
        return nil, result.Error
    }
    
    return model.ToAPI(), nil
}

func (s *DeviceStoreImpl) List(ctx context.Context, orgID uuid.UUID, opts ListOptions) ([]*v1beta1.Device, error) {
    ctx, span := tracing.StartSpan(ctx, "store.Device.List")
    defer span.End()
    
    query := s.db.WithContext(ctx).Where("org_id = ?", orgID)
    
    // Apply label selector
    if opts.LabelSelector != "" {
        query = applyLabelSelector(query, opts.LabelSelector)
    }
    
    // Apply field selector
    if opts.FieldSelector != "" {
        query = applyFieldSelector(query, opts.FieldSelector)
    }
    
    // Apply pagination
    if opts.Continue != "" {
        query = query.Where("name > ?", opts.Continue)
    }
    query = query.Limit(opts.Limit).Order("name ASC")
    
    var models []DeviceModel
    if err := query.Find(&models).Error; err != nil {
        return nil, err
    }
    
    devices := make([]*v1beta1.Device, len(models))
    for i, model := range models {
        devices[i] = model.ToAPI()
    }
    
    return devices, nil
}
```

### Label Selector Implementation

```go
func applyLabelSelector(query *gorm.DB, selector string) *gorm.DB {
    requirements := parseSelector(selector)
    
    for _, req := range requirements {
        switch req.Operator {
        case "=", "==":
            query = query.Where("labels @> ?", 
                fmt.Sprintf(`{"%s": "%s"}`, req.Key, req.Value))
        case "!=":
            query = query.Where("NOT (labels @> ?)", 
                fmt.Sprintf(`{"%s": "%s"}`, req.Key, req.Value))
        case "in":
            // labels->'key' IN ('val1', 'val2')
            query = query.Where("labels->? IN (?)", req.Key, req.Values)
        case "notin":
            query = query.Where("labels->? NOT IN (?)", req.Key, req.Values)
        case "exists":
            query = query.Where("labels ? ?", req.Key)
        case "!":
            query = query.Where("NOT (labels ? ?)", req.Key)
        }
    }
    
    return query
}
```

---

## Queue System

### Queue Provider Interface

```go
type Provider interface {
    // Task queues
    Enqueue(ctx context.Context, queue string, task interface{}) error
    Dequeue(ctx context.Context, queue string) (<-chan Task, error)
    
    // Pub/sub
    Publish(ctx context.Context, channel string, message interface{}) error
    Subscribe(ctx context.Context, channel string) (<-chan Message, error)
    
    // Distributed locks
    Lock(ctx context.Context, key string, ttl time.Duration) (Lock, error)
    
    // Metrics
    GetQueueDepth(ctx context.Context, queue string) (int64, error)
    GetQueueLatency(ctx context.Context, queue string) (time.Duration, error)
}
```

### Redis Implementation

```go
type RedisProvider struct {
    client    *redis.Client
    processID string
    log       *logrus.Entry
}

func (p *RedisProvider) Lock(ctx context.Context, key string, ttl time.Duration) (Lock, error) {
    lockKey := "lock:" + key
    lockValue := p.processID + ":" + uuid.New().String()
    
    // Try to acquire lock
    ok, err := p.client.SetNX(ctx, lockKey, lockValue, ttl).Result()
    if err != nil {
        return nil, err
    }
    if !ok {
        return nil, ErrLockNotAcquired
    }
    
    // Return lock handle
    return &RedisLock{
        client: p.client,
        key:    lockKey,
        value:  lockValue,
        ttl:    ttl,
    }, nil
}

type RedisLock struct {
    client *redis.Client
    key    string
    value  string
    ttl    time.Duration
    ctx    context.Context
    cancel context.CancelFunc
}

func (l *RedisLock) Extend(ctx context.Context) error {
    // Extend TTL if we still own the lock
    script := `
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("pexpire", KEYS[1], ARGV[2])
        else
            return 0
        end
    `
    return l.client.Eval(ctx, script, []string{l.key}, l.value, l.ttl.Milliseconds()).Err()
}

func (l *RedisLock) Release(ctx context.Context) error {
    // Release lock only if we still own it
    script := `
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("del", KEYS[1])
        else
            return 0
        end
    `
    return l.client.Eval(ctx, script, []string{l.key}, l.value).Err()
}
```

---

## Rollout Management

### Rollout Strategy

```go
type RolloutStrategy struct {
    MaxUnavailable intstr.IntOrString  // Max devices updating simultaneously
    MaxSurge       intstr.IntOrString  // Max devices beyond desired count
    ProgressDeadlineSeconds int32      // Max time for rollout
}

func calculateMaxUnavailable(total int, maxUnavailable intstr.IntOrString) int {
    if maxUnavailable.Type == intstr.Int {
        return maxUnavailable.IntVal
    }
    // Percentage
    percent := maxUnavailable.StrVal
    pct, _ := strconv.Atoi(strings.TrimSuffix(percent, "%"))
    return (total * pct) / 100
}
```

### Rollout State Machine

```
┌──────────────┐
│   Pending    │  Initial state
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  InProgress  │  Rolling out to devices
└──────┬───────┘
       │
       ├─────────────────┐
       ▼                 ▼
┌──────────────┐  ┌──────────────┐
│  Completed   │  │    Failed    │
└──────────────┘  └──────────────┘
       │                 │
       │                 ▼
       │          ┌──────────────┐
       │          │  RollingBack │
       │          └──────────────┘
       │                 │
       └─────────────────┘
```

---

## Resource Rendering

### Template Priority

1. **Device Spec** (Highest priority)
2. **Fleet Templates** (Applied in chronological order)
3. **Default Values** (Lowest priority)

### Rendering Algorithm

```go
func renderDevice(device *Device, fleets []*Fleet) *RenderedSpec {
    rendered := &RenderedSpec{}
    
    // Start with defaults
    applyDefaults(rendered)
    
    // Apply fleet templates (oldest first)
    sort.Slice(fleets, byCreationTime)
    for _, fleet := range fleets {
        mergeTemplate(rendered, fleet.Spec.Template)
    }
    
    // Apply device-specific config (overrides fleets)
    if device.Spec != nil {
        mergeSpec(rendered, device.Spec)
    }
    
    // Resolve references
    resolveReferences(rendered)
    
    return rendered
}
```

---

## Certificate Management

### Certificate Lifecycle

```
1. Device generates key pair in TPM
2. Device creates CSR with TPM attestation
3. Device submits EnrollmentRequest
4. Admin/automation approves EnrollmentRequest
5. Service verifies TPM attestation
6. Service creates CSR resource
7. Service signs certificate
8. Device retrieves certificate
9. Device uses certificate for mTLS
10. Before expiry, device generates new CSR
11. Service auto-approves renewal (same TPM)
12. Repeat from step 7
```

### Auto-Renewal

```go
type CertManager struct {
    ca           *crypto.CA
    store        store.Store
    renewBefore  time.Duration
}

func (m *CertManager) CheckRenewal(ctx context.Context) {
    devices, _ := m.store.Device().List(ctx, ListOptions{})
    
    for _, device := range devices {
        cert := device.Status.Certificate
        if cert == nil {
            continue
        }
        
        expiresAt := cert.NotAfter
        renewAt := expiresAt.Add(-m.renewBefore)
        
        if time.Now().After(renewAt) {
            m.renewCertificate(ctx, device)
        }
    }
}
```

---

## Observability

### Metrics

**Key Metrics Exposed**:
- `flightctl_api_requests_total` - HTTP request count
- `flightctl_api_request_duration_seconds` - Request latency
- `flightctl_devices_total` - Total devices by status
- `flightctl_queue_depth` - Task queue depth
- `flightctl_queue_processing_duration_seconds` - Task processing time
- `flightctl_rollout_progress` - Rollout completion percentage

### Tracing

**OpenTelemetry Integration**:

```go
func InitTracer(log *logrus.Entry, cfg *config.Config, serviceName string) func(context.Context) error {
    exporter, err := otlptracegrpc.New(context.Background(),
        otlptracegrpc.WithEndpoint(cfg.Tracing.Endpoint),
        otlptracegrpc.WithInsecure(),
    )
    
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceName(serviceName),
        )),
    )
    
    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.TraceContext{})
    
    return tp.Shutdown
}
```

---

## Performance Considerations

### Database Optimization

**Indexes**:
```sql
-- Device queries
CREATE INDEX idx_devices_org_name ON devices(org_id, name);
CREATE INDEX idx_devices_labels ON devices USING GIN(labels);
CREATE INDEX idx_devices_owner ON devices(owner);

-- Fleet queries
CREATE INDEX idx_fleets_org_name ON fleets(org_id, name);

-- Status queries
CREATE INDEX idx_devices_last_seen ON devices(last_seen);
CREATE INDEX idx_devices_updated ON devices((status->>'updated'));
```

**Connection Pooling**:
```go
db.SetMaxOpenConns(100)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(time.Hour)
```

### Caching Strategy

**Redis Cache for Hot Data**:
- Rendered device specs (TTL: 5 minutes)
- Fleet templates (TTL: 10 minutes)
- User permissions (TTL: 5 minutes)

### Horizontal Scaling

**API Server**: Stateless, scale based on request rate
**Workers**: Scale based on queue depth
**Database**: Read replicas for read-heavy workloads

---

## Appendix

### Further Reading

- [Architecture Overview](ARCHITECTURE.md)
- [API Documentation](user/references/api-resources.md)
- [Development Workflow](developer/WORKFLOW.md)

