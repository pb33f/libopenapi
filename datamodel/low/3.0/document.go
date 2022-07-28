package v3

type Document struct {
    Version    string
    Info       Info
    Servers    []Server
    Components Components
}
