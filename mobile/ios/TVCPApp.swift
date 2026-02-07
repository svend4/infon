import SwiftUI

@main
struct TVCPApp: App {
    @StateObject private var callManager = CallManager.shared

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(callManager)
        }
    }
}

// Main Content View
struct ContentView: View {
    @EnvironmentObject var callManager: CallManager
    @State private var remoteAddress: String = ""
    @State private var showSettings = false

    var body: some View {
        NavigationView {
            VStack(spacing: 20) {
                // App Title
                Text("TVCP Mobile")
                    .font(.largeTitle)
                    .fontWeight(.bold)

                // Connection Status
                StatusBadge(status: callManager.connectionStatus)

                // Address Input
                if callManager.connectionStatus == .disconnected {
                    VStack(spacing: 10) {
                        TextField("Enter remote address", text: $remoteAddress)
                            .textFieldStyle(RoundedBorderTextFieldStyle())
                            .autocapitalization(.none)
                            .padding(.horizontal)

                        Button(action: {
                            callManager.connect(to: remoteAddress)
                        }) {
                            Text("Connect")
                                .font(.headline)
                                .foregroundColor(.white)
                                .frame(maxWidth: .infinity)
                                .padding()
                                .background(Color.blue)
                                .cornerRadius(10)
                        }
                        .padding(.horizontal)
                        .disabled(remoteAddress.isEmpty)
                    }
                }

                // Active Call View
                if callManager.connectionStatus == .connected {
                    ActiveCallView()
                }

                Spacer()

                // Settings Button
                Button(action: {
                    showSettings = true
                }) {
                    Image(systemName: "gear")
                        .font(.title2)
                        .foregroundColor(.gray)
                }
                .sheet(isPresented: $showSettings) {
                    SettingsView()
                }
            }
            .padding()
            .navigationBarTitleDisplayMode(.inline)
        }
    }
}

// Connection Status Badge
struct StatusBadge: View {
    let status: ConnectionStatus

    var body: some View {
        HStack {
            Circle()
                .fill(statusColor)
                .frame(width: 10, height: 10)

            Text(statusText)
                .font(.subheadline)
                .foregroundColor(.secondary)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 6)
        .background(Color.gray.opacity(0.1))
        .cornerRadius(20)
    }

    var statusColor: Color {
        switch status {
        case .disconnected:
            return .red
        case .connecting:
            return .orange
        case .connected:
            return .green
        }
    }

    var statusText: String {
        switch status {
        case .disconnected:
            return "Disconnected"
        case .connecting:
            return "Connecting..."
        case .connected:
            return "Connected"
        }
    }
}

// Active Call View
struct ActiveCallView: View {
    @EnvironmentObject var callManager: CallManager
    @State private var isMuted = false
    @State private var isCameraOff = false
    @State private var showChat = false

    var body: some View {
        VStack(spacing: 20) {
            // Video Preview
            VideoPreviewView(captureManager: callManager.videoCaptureManager)
                .frame(height: 400)
                .background(Color.black)
                .cornerRadius(15)
                .overlay(
                    RoundedRectangle(cornerRadius: 15)
                        .stroke(Color.blue, lineWidth: 2)
                )

            // Call Statistics
            CallStatsView(stats: callManager.callStats)

            // Control Buttons
            HStack(spacing: 30) {
                // Mute Button
                ControlButton(
                    icon: isMuted ? "mic.slash.fill" : "mic.fill",
                    color: isMuted ? .red : .blue
                ) {
                    isMuted.toggle()
                    callManager.setMuted(isMuted)
                }

                // Camera Button
                ControlButton(
                    icon: isCameraOff ? "video.slash.fill" : "video.fill",
                    color: isCameraOff ? .red : .blue
                ) {
                    isCameraOff.toggle()
                    callManager.setCameraEnabled(!isCameraOff)
                }

                // Chat Button
                ControlButton(
                    icon: "message.fill",
                    color: .green
                ) {
                    showChat = true
                }

                // Hang Up Button
                ControlButton(
                    icon: "phone.down.fill",
                    color: .red
                ) {
                    callManager.disconnect()
                }
            }
            .padding()
            .sheet(isPresented: $showChat) {
                ChatView()
            }
        }
    }
}

// Control Button Component
struct ControlButton: View {
    let icon: String
    let color: Color
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Image(systemName: icon)
                .font(.title2)
                .foregroundColor(.white)
                .frame(width: 60, height: 60)
                .background(color)
                .clipShape(Circle())
                .shadow(radius: 5)
        }
    }
}

// Call Statistics View
struct CallStatsView: View {
    let stats: CallStatistics

    var body: some View {
        VStack(spacing: 8) {
            StatRow(label: "Duration", value: stats.durationString)
            StatRow(label: "Bitrate", value: "\(stats.bitrate) kbps")
            StatRow(label: "Packet Loss", value: String(format: "%.1f%%", stats.packetLoss))
            StatRow(label: "RTT", value: "\(stats.rtt) ms")
        }
        .font(.caption)
        .foregroundColor(.secondary)
        .padding()
        .background(Color.gray.opacity(0.1))
        .cornerRadius(10)
    }
}

struct StatRow: View {
    let label: String
    let value: String

    var body: some View {
        HStack {
            Text(label + ":")
            Spacer()
            Text(value)
                .fontWeight(.semibold)
        }
    }
}

// Video Preview View
struct VideoPreviewView: UIViewRepresentable {
    let captureManager: VideoCaptureManager?

    func makeUIView(context: Context) -> UIView {
        let view = UIView(frame: .zero)
        view.backgroundColor = .black

        if let captureManager = captureManager {
            captureManager.setupPreview(in: view)
        }

        return view
    }

    func updateUIView(_ uiView: UIView, context: Context) {
        // Update if needed
    }
}

// Chat View
struct ChatView: View {
    @EnvironmentObject var callManager: CallManager
    @State private var messageText = ""
    @Environment(\.presentationMode) var presentationMode

    var body: some View {
        NavigationView {
            VStack {
                // Messages List
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 10) {
                        ForEach(callManager.chatMessages) { message in
                            ChatBubble(message: message)
                        }
                    }
                    .padding()
                }

                // Message Input
                HStack {
                    TextField("Type a message...", text: $messageText)
                        .textFieldStyle(RoundedBorderTextFieldStyle())

                    Button(action: sendMessage) {
                        Image(systemName: "paperplane.fill")
                            .foregroundColor(.blue)
                    }
                    .disabled(messageText.isEmpty)
                }
                .padding()
            }
            .navigationTitle("Chat")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") {
                        presentationMode.wrappedValue.dismiss()
                    }
                }
            }
        }
    }

    func sendMessage() {
        callManager.sendChatMessage(messageText)
        messageText = ""
    }
}

// Chat Bubble
struct ChatBubble: View {
    let message: ChatMessage

    var body: some View {
        HStack {
            if message.isSent {
                Spacer()
            }

            VStack(alignment: message.isSent ? .trailing : .leading, spacing: 4) {
                Text(message.text)
                    .padding(10)
                    .background(message.isSent ? Color.blue : Color.gray.opacity(0.2))
                    .foregroundColor(message.isSent ? .white : .primary)
                    .cornerRadius(15)

                Text(message.timestamp, style: .time)
                    .font(.caption2)
                    .foregroundColor(.secondary)
            }

            if !message.isSent {
                Spacer()
            }
        }
    }
}

// Settings View
struct SettingsView: View {
    @EnvironmentObject var callManager: CallManager
    @Environment(\.presentationMode) var presentationMode

    @State private var sampleRate: Int = 16000
    @State private var bitrate: Int = 128
    @State private var enableVAD = true
    @State private var enableNoiseSuppression = true

    var body: some View {
        NavigationView {
            Form {
                Section(header: Text("Audio Settings")) {
                    Picker("Sample Rate", selection: $sampleRate) {
                        Text("8 kHz").tag(8000)
                        Text("16 kHz").tag(16000)
                        Text("44.1 kHz").tag(44100)
                        Text("48 kHz").tag(48000)
                    }

                    Stepper("Bitrate: \(bitrate) kbps", value: $bitrate, in: 32...320, step: 32)

                    Toggle("Voice Activity Detection", isOn: $enableVAD)
                    Toggle("Noise Suppression", isOn: $enableNoiseSuppression)
                }

                Section(header: Text("Video Settings")) {
                    Picker("Resolution", selection: $sampleRate) {
                        Text("640x480").tag(480)
                        Text("1280x720").tag(720)
                        Text("1920x1080").tag(1080)
                    }

                    Stepper("FPS: \(bitrate / 8)", value: $bitrate, in: 15...30, step: 5)
                }

                Section(header: Text("Network")) {
                    Toggle("Use TURN Server", isOn: $enableVAD)
                    Toggle("Adaptive Bitrate", isOn: $enableNoiseSuppression)
                }

                Section {
                    Button("Save Settings") {
                        saveSettings()
                        presentationMode.wrappedValue.dismiss()
                    }
                }
            }
            .navigationTitle("Settings")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Cancel") {
                        presentationMode.wrappedValue.dismiss()
                    }
                }
            }
        }
    }

    func saveSettings() {
        // Save settings to CallManager
        callManager.updateSettings(
            sampleRate: sampleRate,
            bitrate: bitrate,
            enableVAD: enableVAD,
            enableNoiseSuppression: enableNoiseSuppression
        )
    }
}

// Connection Status Enum
enum ConnectionStatus {
    case disconnected
    case connecting
    case connected
}

// Chat Message Model
struct ChatMessage: Identifiable {
    let id = UUID()
    let text: String
    let timestamp: Date
    let isSent: Bool
}

// Call Statistics Model
struct CallStatistics {
    var duration: TimeInterval = 0
    var bitrate: Int = 0
    var packetLoss: Double = 0.0
    var rtt: Int = 0

    var durationString: String {
        let minutes = Int(duration) / 60
        let seconds = Int(duration) % 60
        return String(format: "%02d:%02d", minutes, seconds)
    }
}
