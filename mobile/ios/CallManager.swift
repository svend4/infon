import Foundation
import AVFoundation
import Network
import Combine

// Call Manager - координирует все компоненты звонка
class CallManager: ObservableObject {
    static let shared = CallManager()

    // Published Properties
    @Published var connectionStatus: ConnectionStatus = .disconnected
    @Published var chatMessages: [ChatMessage] = []
    @Published var callStats: CallStatistics = CallStatistics()

    // Components
    var audioCaptureManager: AudioCaptureManager?
    var videoCaptureManager: VideoCaptureManager?
    var networkManager: NetworkManager?
    var codecManager: CodecManager?

    // Private Properties
    private var callStartTime: Date?
    private var statsTimer: Timer?
    private var cancellables = Set<AnyCancelable>()

    init() {
        setupComponents()
    }

    // MARK: - Setup

    func setupComponents() {
        audioCaptureManager = AudioCaptureManager()
        videoCaptureManager = VideoCaptureManager()
        networkManager = NetworkManager()
        codecManager = CodecManager()

        // Setup network callbacks
        networkManager?.onDataReceived = { [weak self] data in
            self?.handleReceivedData(data)
        }

        networkManager?.onConnectionStateChanged = { [weak self] state in
            DispatchQueue.main.async {
                self?.connectionStatus = state
            }
        }
    }

    // MARK: - Connection

    func connect(to address: String) {
        print("Connecting to: \(address)")

        connectionStatus = .connecting

        // Parse address (format: "hostname:port")
        let components = address.split(separator: ":")
        guard components.count == 2,
              let port = UInt16(components[1]) else {
            print("Invalid address format")
            connectionStatus = .disconnected
            return
        }

        let hostname = String(components[0])

        // Start network connection
        networkManager?.connect(host: hostname, port: NWEndpoint.Port(rawValue: port)!) { [weak self] success in
            if success {
                self?.startCall()
            } else {
                DispatchQueue.main.async {
                    self?.connectionStatus = .disconnected
                }
            }
        }
    }

    func disconnect() {
        print("Disconnecting...")

        stopCall()
        networkManager?.disconnect()

        DispatchQueue.main.async {
            self.connectionStatus = .disconnected
            self.callStartTime = nil
            self.chatMessages.removeAll()
        }
    }

    // MARK: - Call Control

    private func startCall() {
        print("Starting call...")

        DispatchQueue.main.async {
            self.connectionStatus = .connected
            self.callStartTime = Date()
        }

        // Start audio capture
        audioCaptureManager?.startCapture { [weak self] audioData in
            self?.handleCapturedAudio(audioData)
        }

        // Start video capture
        videoCaptureManager?.startCapture { [weak self] videoFrame in
            self?.handleCapturedVideo(videoFrame)
        }

        // Start statistics timer
        startStatsTimer()
    }

    private func stopCall() {
        print("Stopping call...")

        audioCaptureManager?.stopCapture()
        videoCaptureManager?.stopCapture()
        statsTimer?.invalidate()
        statsTimer = nil
    }

    // MARK: - Audio/Video Handling

    private func handleCapturedAudio(_ audioData: Data) {
        // Encode audio
        guard let encodedData = codecManager?.encodeAudio(audioData) else {
            return
        }

        // Send over network
        networkManager?.send(encodedData, type: .audio)
    }

    private func handleCapturedVideo(_ videoFrame: CVPixelBuffer) {
        // Convert to data
        guard let imageData = videoCaptureManager?.convertPixelBufferToData(videoFrame) else {
            return
        }

        // Encode video
        guard let encodedData = codecManager?.encodeVideo(imageData) else {
            return
        }

        // Send over network
        networkManager?.send(encodedData, type: .video)
    }

    private func handleReceivedData(_ data: Data) {
        // Parse packet header to determine type
        guard data.count > 4 else { return }

        let packetType = data[0]

        switch packetType {
        case 0x01: // Audio packet
            handleReceivedAudio(data)
        case 0x02: // Video packet
            handleReceivedVideo(data)
        case 0x03: // Chat message
            handleReceivedChatMessage(data)
        default:
            print("Unknown packet type: \(packetType)")
        }
    }

    private func handleReceivedAudio(_ data: Data) {
        // Decode audio
        guard let decodedData = codecManager?.decodeAudio(data) else {
            return
        }

        // Play audio
        audioCaptureManager?.playback(decodedData)
    }

    private func handleReceivedVideo(_ data: Data) {
        // Decode video
        guard let decodedData = codecManager?.decodeVideo(data) else {
            return
        }

        // Display video (placeholder)
        // In full implementation, would update UI with decoded frame
    }

    private func handleReceivedChatMessage(_ data: Data) {
        // Extract message text
        guard let messageText = String(data: data.dropFirst(), encoding: .utf8) else {
            return
        }

        DispatchQueue.main.async {
            let message = ChatMessage(
                text: messageText,
                timestamp: Date(),
                isSent: false
            )
            self.chatMessages.append(message)
        }
    }

    // MARK: - Controls

    func setMuted(_ muted: Bool) {
        audioCaptureManager?.setMuted(muted)
    }

    func setCameraEnabled(_ enabled: Bool) {
        if enabled {
            videoCaptureManager?.resumeCapture()
        } else {
            videoCaptureManager?.pauseCapture()
        }
    }

    func sendChatMessage(_ text: String) {
        // Create chat message
        let message = ChatMessage(
            text: text,
            timestamp: Date(),
            isSent: true
        )

        DispatchQueue.main.async {
            self.chatMessages.append(message)
        }

        // Send over network
        var data = Data([0x03]) // Chat packet type
        if let textData = text.data(using: .utf8) {
            data.append(textData)
        }

        networkManager?.send(data, type: .chat)
    }

    // MARK: - Settings

    func updateSettings(sampleRate: Int, bitrate: Int, enableVAD: Bool, enableNoiseSuppression: Bool) {
        audioCaptureManager?.updateSettings(
            sampleRate: sampleRate,
            enableVAD: enableVAD,
            enableNoiseSuppression: enableNoiseSuppression
        )

        codecManager?.updateSettings(bitrate: bitrate)
    }

    // MARK: - Statistics

    private func startStatsTimer() {
        statsTimer = Timer.scheduledTimer(withTimeInterval: 1.0, repeats: true) { [weak self] _ in
            self?.updateStatistics()
        }
    }

    private func updateStatistics() {
        guard let startTime = callStartTime else { return }

        DispatchQueue.main.async {
            self.callStats.duration = Date().timeIntervalSince(startTime)

            // Get network stats
            if let networkStats = self.networkManager?.getStatistics() {
                self.callStats.bitrate = networkStats.bitrate
                self.callStats.packetLoss = networkStats.packetLoss
                self.callStats.rtt = networkStats.rtt
            }
        }
    }
}

// MARK: - Audio Capture Manager

class AudioCaptureManager {
    private var audioEngine: AVAudioEngine?
    private var inputNode: AVAudioInputNode?
    private var isMuted = false
    private var sampleRate: Double = 16000
    private var onAudioCaptured: ((Data) -> Void)?

    func startCapture(onAudioCaptured: @escaping (Data) -> Void) {
        self.onAudioCaptured = onAudioCaptured

        audioEngine = AVAudioEngine()
        guard let audioEngine = audioEngine else { return }

        inputNode = audioEngine.inputNode

        // Configure audio format
        let format = AVAudioFormat(
            commonFormat: .pcmFormatInt16,
            sampleRate: sampleRate,
            channels: 1,
            interleaved: false
        )!

        // Install tap
        inputNode?.installTap(onBus: 0, bufferSize: 320, format: format) { [weak self] buffer, time in
            guard let self = self, !self.isMuted else { return }
            self.processAudioBuffer(buffer)
        }

        // Start engine
        do {
            try audioEngine.start()
            print("Audio capture started")
        } catch {
            print("Failed to start audio engine: \(error)")
        }
    }

    func stopCapture() {
        inputNode?.removeTap(onBus: 0)
        audioEngine?.stop()
        audioEngine = nil
        inputNode = nil
    }

    func setMuted(_ muted: Bool) {
        isMuted = muted
    }

    func updateSettings(sampleRate: Int, enableVAD: Bool, enableNoiseSuppression: Bool) {
        self.sampleRate = Double(sampleRate)
        // Would restart capture with new settings
    }

    func playback(_ audioData: Data) {
        // Playback implementation (simplified)
        // In full implementation, would use AVAudioPlayerNode
    }

    private func processAudioBuffer(_ buffer: AVAudioPCMBuffer) {
        guard let channelData = buffer.int16ChannelData else { return }

        let frameLength = Int(buffer.frameLength)
        let data = Data(bytes: channelData[0], count: frameLength * MemoryLayout<Int16>.size)

        onAudioCaptured?(data)
    }
}

// MARK: - Video Capture Manager

class VideoCaptureManager: NSObject, AVCaptureVideoDataOutputSampleBufferDelegate {
    private var captureSession: AVCaptureSession?
    private var previewLayer: AVCaptureVideoPreviewLayer?
    private var videoOutput: AVCaptureVideoDataOutput?
    private var onVideoCaptured: ((CVPixelBuffer) -> Void)?
    private var isPaused = false

    func setupPreview(in view: UIView) {
        guard let captureSession = captureSession else { return }

        previewLayer = AVCaptureVideoPreviewLayer(session: captureSession)
        previewLayer?.videoGravity = .resizeAspectFill
        previewLayer?.frame = view.bounds

        if let previewLayer = previewLayer {
            view.layer.addSublayer(previewLayer)
        }
    }

    func startCapture(onVideoCaptured: @escaping (CVPixelBuffer) -> Void) {
        self.onVideoCaptured = onVideoCaptured

        captureSession = AVCaptureSession()
        guard let captureSession = captureSession else { return }

        captureSession.beginConfiguration()

        // Add video input
        guard let videoDevice = AVCaptureDevice.default(.builtInWideAngleCamera, for: .video, position: .front),
              let videoInput = try? AVCaptureDeviceInput(device: videoDevice),
              captureSession.canAddInput(videoInput) else {
            print("Failed to add video input")
            return
        }

        captureSession.addInput(videoInput)

        // Add video output
        videoOutput = AVCaptureVideoDataOutput()
        videoOutput?.setSampleBufferDelegate(self, queue: DispatchQueue(label: "videoQueue"))

        if let videoOutput = videoOutput,
           captureSession.canAddOutput(videoOutput) {
            captureSession.addOutput(videoOutput)
        }

        captureSession.commitConfiguration()

        // Start session
        DispatchQueue.global(qos: .userInitiated).async {
            captureSession.startRunning()
        }

        print("Video capture started")
    }

    func stopCapture() {
        captureSession?.stopRunning()
        captureSession = nil
    }

    func pauseCapture() {
        isPaused = true
    }

    func resumeCapture() {
        isPaused = false
    }

    func convertPixelBufferToData(_ pixelBuffer: CVPixelBuffer) -> Data? {
        // Convert CVPixelBuffer to Data (simplified)
        // Full implementation would use proper image conversion
        return Data()
    }

    // AVCaptureVideoDataOutputSampleBufferDelegate
    func captureOutput(_ output: AVCaptureOutput, didOutput sampleBuffer: CMSampleBuffer, from connection: AVCaptureConnection) {
        guard !isPaused,
              let pixelBuffer = CMSampleBufferGetImageBuffer(sampleBuffer) else {
            return
        }

        onVideoCaptured?(pixelBuffer)
    }
}

// MARK: - Network Manager

class NetworkManager {
    private var connection: NWConnection?
    private var receiveQueue = DispatchQueue(label: "receiveQueue")

    var onDataReceived: ((Data) -> Void)?
    var onConnectionStateChanged: ((ConnectionStatus) -> Void)?

    func connect(host: String, port: NWEndpoint.Port, completion: @escaping (Bool) -> Void) {
        let endpoint = NWEndpoint.hostPort(host: NWEndpoint.Host(host), port: port)
        connection = NWConnection(to: endpoint, using: .udp)

        connection?.stateUpdateHandler = { [weak self] state in
            switch state {
            case .ready:
                print("Network connection ready")
                self?.onConnectionStateChanged?(.connected)
                self?.startReceiving()
                completion(true)
            case .failed(let error):
                print("Network connection failed: \(error)")
                self?.onConnectionStateChanged?(.disconnected)
                completion(false)
            default:
                break
            }
        }

        connection?.start(queue: receiveQueue)
    }

    func disconnect() {
        connection?.cancel()
        connection = nil
    }

    func send(_ data: Data, type: PacketType) {
        connection?.send(content: data, completion: .contentProcessed { error in
            if let error = error {
                print("Send error: \(error)")
            }
        })
    }

    private func startReceiving() {
        connection?.receiveMessage { [weak self] data, context, isComplete, error in
            if let data = data {
                self?.onDataReceived?(data)
            }

            // Continue receiving
            self?.startReceiving()
        }
    }

    func getStatistics() -> NetworkStatistics {
        // Return mock statistics
        return NetworkStatistics(bitrate: 128, packetLoss: 0.5, rtt: 50)
    }
}

// MARK: - Codec Manager

class CodecManager {
    private var audioBitrate = 128000
    private var videoBitrate = 1000000

    func encodeAudio(_ data: Data) -> Data? {
        // Simplified audio encoding (Opus would be used in production)
        return data
    }

    func decodeAudio(_ data: Data) -> Data? {
        // Simplified audio decoding
        return data.dropFirst() // Remove packet type byte
    }

    func encodeVideo(_ data: Data) -> Data? {
        // Simplified video encoding (H.264 would be used in production)
        return data
    }

    func decodeVideo(_ data: Data) -> Data? {
        // Simplified video decoding
        return data.dropFirst() // Remove packet type byte
    }

    func updateSettings(bitrate: Int) {
        self.audioBitrate = bitrate * 1000
    }
}

// MARK: - Supporting Types

enum PacketType {
    case audio
    case video
    case chat
}

struct NetworkStatistics {
    let bitrate: Int
    let packetLoss: Double
    let rtt: Int
}
