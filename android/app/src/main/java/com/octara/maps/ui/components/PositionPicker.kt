package com.octara.maps.ui.components

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.MyLocation
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.octara.maps.types.ParentStop
import com.octara.maps.types.Position
import com.octara.maps.utils.Api
import com.octara.maps.utils.LocationHelper
import kotlinx.coroutines.delay

@Composable
fun PositionPicker(
    onDismiss: () -> Unit,
    onSelectPosition: (position: Position, displayName: String) -> Unit,
    modifier: Modifier = Modifier
) {
    BackHandler {
        onDismiss()
    }

    var query by remember { mutableStateOf("") }
    var stops by remember { mutableStateOf<List<ParentStop>>(emptyList()) }
    var isLoading by remember { mutableStateOf(false) }

    val context = LocalContext.current
    var currentCoords by remember { mutableStateOf(Pair(48.8566f, 2.3522f)) }

    LaunchedEffect(Unit) {
        val loc = LocationHelper(context).getCurrentLocation()
        if (loc != null) {
            currentCoords = loc
        }
    }

    LaunchedEffect(query) {
        if (query.length < 2) {
            isLoading = false
            stops = emptyList()
            return@LaunchedEffect
        }

        delay(300)
        isLoading = true
        try {
            val result = Api().searchStopsByName(
                name = query,
                limit = 20,
                cpos = currentCoords
            )
            stops = result ?: emptyList()
        } catch (_: Exception) {
            stops = emptyList()
        } finally {
            isLoading = false
        }
    }

    Surface(
        modifier = modifier.fillMaxSize(),
        color = MaterialTheme.colorScheme.surface
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .statusBarsPadding()
        ) {
            AppSearchBar(
                query = query,
                onQueryChange = { query = it },
                onSearch = {},
                active = true,
                autoFocus = true,
                onBack = onDismiss,
                placeholder = "Rechercher un arrêt...",
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                isLoading = isLoading
            )

            LazyColumn(
                modifier = Modifier
                    .fillMaxWidth()
                    .weight(1f)
                    .padding(horizontal = 16.dp)
            ) {
                item {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(12.dp))
                            .clickable {
                                onSelectPosition(Position.CurrentLocation, "Ma position")
                            }
                            .padding(vertical = 12.dp, horizontal = 4.dp),
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Box(
                            modifier = Modifier
                                .size(40.dp)
                                .clip(RoundedCornerShape(10.dp))
                                .background(Color(0xFFEBF3FF)),
                            contentAlignment = Alignment.Center
                        ) {
                            Icon(
                                imageVector = Icons.Default.MyLocation,
                                contentDescription = "Ma position",
                                tint = Color(0xFF007AFF),
                                modifier = Modifier.size(22.dp)
                            )
                        }
                        Column {
                            Text(
                                text = "Ma position",
                                style = MaterialTheme.typography.bodyLarge,
                                fontWeight = FontWeight.SemiBold,
                                color = Color(0xFF007AFF)
                            )
                            Text(
                                text = "Position GPS actuelle",
                                style = MaterialTheme.typography.bodySmall,
                                color = Color(0xFF888888)
                            )
                        }
                    }

                    HorizontalDivider(
                        modifier = Modifier.padding(vertical = 8.dp),
                        thickness = 0.5.dp,
                        color = Color(0xFFE0E0E0)
                    )
                }

                items(stops) { stop ->
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(12.dp))
                            .clickable {
                                onSelectPosition(Position.ByStop(stop.id), stop.name)
                            }
                            .padding(vertical = 8.dp, horizontal = 4.dp),
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Box(
                            modifier = Modifier
                                .size(40.dp)
                                .clip(RoundedCornerShape(10.dp))
                                .background(MaterialTheme.colorScheme.primary.copy(alpha = 0.1f)),
                            contentAlignment = Alignment.Center
                        ) {
                            Icon(
                                imageVector = getStopIconResource(stop.type),
                                contentDescription = "Icône de transport",
                                tint = MaterialTheme.colorScheme.primary
                            )
                        }
                        Text(
                            text = stop.name,
                            style = MaterialTheme.typography.bodyLarge,
                            fontWeight = FontWeight.Medium,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                    }
                }
            }
        }
    }
}