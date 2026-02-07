package com.tvcp.mobile

import android.Manifest
import android.content.pm.PackageManager
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.content.ContextCompat
import androidx.lifecycle.viewmodel.compose.viewModel
import java.text.SimpleDateFormat
import java.util.*

class MainActivity : ComponentActivity() {
    private lateinit var callManager: CallManager

    private val permissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions()
    ) { permissions ->
        val allGranted = permissions.values.all { it }
        if (allGranted) {
            initializeCallManager()
        } else {
            // Show error message
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Request permissions
        requestPermissions()

        setContent {
            TVCPTheme {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = MaterialTheme.colorScheme.background
                ) {
                    MainScreen(callManager = callManager)
                }
            }
        }
    }

    private fun requestPermissions() {
        val permissions = arrayOf(
            Manifest.permission.CAMERA,
            Manifest.permission.RECORD_AUDIO,
            Manifest.permission.INTERNET
        )

        val allGranted = permissions.all {
            ContextCompat.checkSelfPermission(this, it) == PackageManager.PERMISSION_GRANTED
        }

        if (allGranted) {
            initializeCallManager()
        } else {
            permissionLauncher.launch(permissions)
        }
    }

    private fun initializeCallManager() {
        callManager = CallManager(this)
    }
}

// Material3 Theme
@Composable
fun TVCPTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = lightColorScheme(
            primary = Color(0xFF2196F3),
            secondary = Color(0xFF03DAC6),
            error = Color(0xFFB00020)
        ),
        content = content
    )
}

// Main Screen
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MainScreen(callManager: CallManager?) {
    val viewModel: CallViewModel = viewModel()
    var showSettings by remember { mutableStateOf(false) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("TVCP Mobile") },
                actions = {
                    IconButton(onClick = { showSettings = true }) {
                        Icon(Icons.Default.Settings, "Settings")
                    }
                }
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            // Connection Status
            ConnectionStatusBadge(status = viewModel.connectionStatus.value)

            Spacer(modifier = Modifier.height(24.dp))

            // Connection UI or Active Call
            when (viewModel.connectionStatus.value) {
                ConnectionStatus.DISCONNECTED -> {
                    ConnectionView(
                        onConnect = { address ->
                            callManager?.connect(address)
                            viewModel.updateStatus(ConnectionStatus.CONNECTING)
                        }
                    )
                }
                ConnectionStatus.CONNECTING -> {
                    CircularProgressIndicator()
                    Text("Connecting...", modifier = Modifier.padding(top = 16.dp))
                }
                ConnectionStatus.CONNECTED -> {
                    ActiveCallView(
                        callManager = callManager,
                        viewModel = viewModel
                    )
                }
            }
        }

        if (showSettings) {
            SettingsDialog(
                onDismiss = { showSettings = false },
                onSave = { settings ->
                    callManager?.updateSettings(settings)
                    showSettings = false
                }
            )
        }
    }
}

// Connection Status Badge
@Composable
fun ConnectionStatusBadge(status: ConnectionStatus) {
    Row(
        modifier = Modifier
            .padding(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Surface(
            modifier = Modifier.size(12.dp),
            shape = CircleShape,
            color = when (status) {
                ConnectionStatus.DISCONNECTED -> Color.Red
                ConnectionStatus.CONNECTING -> Color(0xFFFFA500) // Orange
                ConnectionStatus.CONNECTED -> Color.Green
            }
        ) {}

        Spacer(modifier = Modifier.width(8.dp))

        Text(
            text = when (status) {
                ConnectionStatus.DISCONNECTED -> "Disconnected"
                ConnectionStatus.CONNECTING -> "Connecting..."
                ConnectionStatus.CONNECTED -> "Connected"
            },
            style = MaterialTheme.typography.bodyMedium
        )
    }
}

// Connection View
@Composable
fun ConnectionView(onConnect: (String) -> Unit) {
    var address by remember { mutableStateOf("") }

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(16.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        OutlinedTextField(
            value = address,
            onValueChange = { address = it },
            label = { Text("Remote Address") },
            placeholder = { Text("hostname:port") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true
        )

        Spacer(modifier = Modifier.height(16.dp))

        Button(
            onClick = { onConnect(address) },
            enabled = address.isNotEmpty(),
            modifier = Modifier.fillMaxWidth()
        ) {
            Text("Connect")
        }
    }
}

// Active Call View
@Composable
fun ActiveCallView(callManager: CallManager?, viewModel: CallViewModel) {
    var isMuted by remember { mutableStateOf(false) }
    var isCameraOff by remember { mutableStateOf(false) }
    var showChat by remember { mutableStateOf(false) }

    Column(
        modifier = Modifier.fillMaxWidth(),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        // Video Preview
        Card(
            modifier = Modifier
                .fillMaxWidth()
                .height(400.dp),
            shape = RoundedCornerShape(16.dp)
        ) {
            AndroidView(
                factory = { context ->
                    callManager?.getVideoView(context) ?: android.view.View(context)
                }
            )
        }

        Spacer(modifier = Modifier.height(16.dp))

        // Call Statistics
        CallStatsCard(stats = viewModel.callStats.value)

        Spacer(modifier = Modifier.height(24.dp))

        // Control Buttons
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceEvenly
        ) {
            // Mute Button
            FloatingActionButton(
                onClick = {
                    isMuted = !isMuted
                    callManager?.setMuted(isMuted)
                },
                containerColor = if (isMuted) Color.Red else MaterialTheme.colorScheme.primary
            ) {
                Icon(
                    if (isMuted) Icons.Default.MicOff else Icons.Default.Mic,
                    "Mute"
                )
            }

            // Camera Button
            FloatingActionButton(
                onClick = {
                    isCameraOff = !isCameraOff
                    callManager?.setCameraEnabled(!isCameraOff)
                },
                containerColor = if (isCameraOff) Color.Red else MaterialTheme.colorScheme.primary
            ) {
                Icon(
                    if (isCameraOff) Icons.Default.VideocamOff else Icons.Default.Videocam,
                    "Camera"
                )
            }

            // Chat Button
            FloatingActionButton(
                onClick = { showChat = true },
                containerColor = Color(0xFF4CAF50) // Green
            ) {
                Icon(Icons.Default.Message, "Chat")
            }

            // Hang Up Button
            FloatingActionButton(
                onClick = {
                    callManager?.disconnect()
                    viewModel.updateStatus(ConnectionStatus.DISCONNECTED)
                },
                containerColor = Color.Red
            ) {
                Icon(Icons.Default.CallEnd, "Hang Up")
            }
        }
    }

    if (showChat) {
        ChatDialog(
            messages = viewModel.chatMessages.value,
            onDismiss = { showChat = false },
            onSendMessage = { message ->
                callManager?.sendChatMessage(message)
                viewModel.addChatMessage(message, true)
            }
        )
    }
}

// Call Statistics Card
@Composable
fun CallStatsCard(stats: CallStatistics) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Column(
            modifier = Modifier.padding(16.dp)
        ) {
            StatRow("Duration", stats.durationString)
            StatRow("Bitrate", "${stats.bitrate} kbps")
            StatRow("Packet Loss", String.format("%.1f%%", stats.packetLoss))
            StatRow("RTT", "${stats.rtt} ms")
        }
    }
}

@Composable
fun StatRow(label: String, value: String) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        Text(text = "$label:", style = MaterialTheme.typography.bodyMedium)
        Text(
            text = value,
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = FontWeight.SemiBold
        )
    }
}

// Chat Dialog
@Composable
fun ChatDialog(
    messages: List<ChatMessage>,
    onDismiss: () -> Unit,
    onSendMessage: (String) -> Unit
) {
    var messageText by remember { mutableStateOf("") }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Chat") },
        text = {
            Column {
                LazyColumn(
                    modifier = Modifier
                        .weight(1f)
                        .fillMaxWidth()
                ) {
                    items(messages) { message ->
                        ChatBubble(message = message)
                    }
                }

                Spacer(modifier = Modifier.height(8.dp))

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    OutlinedTextField(
                        value = messageText,
                        onValueChange = { messageText = it },
                        label = { Text("Message") },
                        modifier = Modifier.weight(1f)
                    )

                    IconButton(
                        onClick = {
                            if (messageText.isNotEmpty()) {
                                onSendMessage(messageText)
                                messageText = ""
                            }
                        }
                    ) {
                        Icon(Icons.Default.Send, "Send")
                    }
                }
            }
        },
        confirmButton = {
            TextButton(onClick = onDismiss) {
                Text("Close")
            }
        }
    )
}

@Composable
fun ChatBubble(message: ChatMessage) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        horizontalArrangement = if (message.isSent) {
            Arrangement.End
        } else {
            Arrangement.Start
        }
    ) {
        Card(
            colors = CardDefaults.cardColors(
                containerColor = if (message.isSent) {
                    MaterialTheme.colorScheme.primary
                } else {
                    MaterialTheme.colorScheme.surfaceVariant
                }
            ),
            shape = RoundedCornerShape(12.dp)
        ) {
            Column(
                modifier = Modifier.padding(8.dp)
            ) {
                Text(
                    text = message.text,
                    color = if (message.isSent) Color.White else Color.Black
                )
                Text(
                    text = SimpleDateFormat("HH:mm", Locale.getDefault())
                        .format(message.timestamp),
                    style = MaterialTheme.typography.bodySmall,
                    color = if (message.isSent) Color.White.copy(alpha = 0.7f) else Color.Gray
                )
            }
        }
    }
}

// Settings Dialog
@Composable
fun SettingsDialog(
    onDismiss: () -> Unit,
    onSave: (CallSettings) -> Unit
) {
    var sampleRate by remember { mutableStateOf(16000) }
    var bitrate by remember { mutableStateOf(128) }
    var enableVAD by remember { mutableStateOf(true) }
    var enableNoiseSuppression by remember { mutableStateOf(true) }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Settings") },
        text = {
            Column {
                Text("Audio Settings", fontWeight = FontWeight.Bold)
                Spacer(modifier = Modifier.height(8.dp))

                // Sample Rate
                Text("Sample Rate: $sampleRate Hz")
                Slider(
                    value = sampleRate.toFloat(),
                    onValueChange = { sampleRate = it.toInt() },
                    valueRange = 8000f..48000f,
                    steps = 3
                )

                // Bitrate
                Text("Bitrate: $bitrate kbps")
                Slider(
                    value = bitrate.toFloat(),
                    onValueChange = { bitrate = it.toInt() },
                    valueRange = 32f..320f
                )

                // VAD
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text("Voice Activity Detection")
                    Switch(
                        checked = enableVAD,
                        onCheckedChange = { enableVAD = it }
                    )
                }

                // Noise Suppression
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text("Noise Suppression")
                    Switch(
                        checked = enableNoiseSuppression,
                        onCheckedChange = { enableNoiseSuppression = it }
                    )
                }
            }
        },
        confirmButton = {
            Button(
                onClick = {
                    onSave(
                        CallSettings(
                            sampleRate = sampleRate,
                            bitrate = bitrate,
                            enableVAD = enableVAD,
                            enableNoiseSuppression = enableNoiseSuppression
                        )
                    )
                }
            ) {
                Text("Save")
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text("Cancel")
            }
        }
    )
}

// Data Classes

enum class ConnectionStatus {
    DISCONNECTED,
    CONNECTING,
    CONNECTED
}

data class CallStatistics(
    val duration: Long = 0,
    val bitrate: Int = 0,
    val packetLoss: Double = 0.0,
    val rtt: Int = 0
) {
    val durationString: String
        get() {
            val minutes = duration / 60
            val seconds = duration % 60
            return String.format("%02d:%02d", minutes, seconds)
        }
}

data class ChatMessage(
    val text: String,
    val timestamp: Date,
    val isSent: Boolean
)

data class CallSettings(
    val sampleRate: Int,
    val bitrate: Int,
    val enableVAD: Boolean,
    val enableNoiseSuppression: Boolean
)
