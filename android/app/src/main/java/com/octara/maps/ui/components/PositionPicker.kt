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
import androidx.compose.material.icons.filled.LocationOn
import androidx.compose.material.icons.filled.MyLocation
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.octara.maps.types.ParentStop
import com.octara.maps.types.Position
import com.octara.maps.utils.Api
import com.octara.maps.utils.LocationHelper
import com.octara.maps.utils.LocationManager
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
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
    var addresses by remember { mutableStateOf<List<Api.AddressResult>>(emptyList()) }
    var isLoading by remember { mutableStateOf(false) }

    val context = LocalContext.current
    val currentCoords by LocationManager.currentLocation.collectAsState()

    LaunchedEffect(Unit) {
        LocationManager.fetchCurrentLocation(context)
    }

    LaunchedEffect(query) {
        if (query.length < 2) {
            isLoading = false
            stops = emptyList()
            addresses = emptyList()
            return@LaunchedEffect
        }

        delay(300)
        isLoading = true
        try {
            val api = Api()
            val (stopsRes, addressesRes) = coroutineScope {
                val s = async { runCatching { api.searchStopsByName(name = query, limit = 15, cpos = currentCoords) }.getOrDefault(emptyList()) }
                val a = async { runCatching { api.searchAddresses(query = query, limit = 10, cpos = currentCoords) }.getOrDefault(emptyList()) }
                s.await() to a.await()
            }
            stops = stopsRes
            addresses = addressesRes
        } catch (_: Exception) {
            stops = emptyList()
            addresses = emptyList()
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
                placeholder = "Rechercher un arrêt ou une adresse...",
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
                    val hasGps = currentCoords != null
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
                                .background(if (hasGps) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceVariant),
                            contentAlignment = Alignment.Center
                        ) {
                            Icon(
                                imageVector = Icons.Default.MyLocation,
                                contentDescription = "Ma position",
                                tint = if (hasGps) MaterialTheme.colorScheme.onPrimaryContainer else MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.size(22.dp)
                            )
                        }
                        Column {
                            Text(
                                text = "Ma position",
                                style = MaterialTheme.typography.bodyLarge,
                                fontWeight = FontWeight.SemiBold,
                                color = if (hasGps) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface
                            )
                            Text(
                                text = if (hasGps) "Position GPS actuelle" else "Position GPS non disponible",
                                style = MaterialTheme.typography.bodySmall,
                                color = if (hasGps) MaterialTheme.colorScheme.onSurfaceVariant else MaterialTheme.colorScheme.error
                            )
                        }
                    }

                    HorizontalDivider(
                        modifier = Modifier.padding(vertical = 8.dp),
                        thickness = 0.5.dp,
                        color = MaterialTheme.colorScheme.outlineVariant
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

                items(addresses) { address ->
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(12.dp))
                            .clickable {
                                onSelectPosition(
                                    Position.Coordinates(address.lat.toDouble(), address.lon.toDouble()),
                                    address.name
                                )
                            }
                            .padding(vertical = 8.dp, horizontal = 4.dp),
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Box(
                            modifier = Modifier
                                .size(40.dp)
                                .clip(RoundedCornerShape(10.dp))
                                .background(MaterialTheme.colorScheme.secondary.copy(alpha = 0.1f)),
                            contentAlignment = Alignment.Center
                        ) {
                            Icon(
                                imageVector = Icons.Default.LocationOn,
                                contentDescription = "Adresse",
                                tint = MaterialTheme.colorScheme.secondary
                            )
                        }
                        Column {
                            Text(
                                text = address.name,
                                style = MaterialTheme.typography.bodyLarge,
                                fontWeight = FontWeight.Medium,
                                color = MaterialTheme.colorScheme.onSurface
                            )
                            val subtitle = listOfNotNull(
                                address.postcode?.takeIf { it.isNotBlank() },
                                address.city?.takeIf { it.isNotBlank() }
                            ).joinToString(" ").ifBlank { address.label }
                            if (subtitle.isNotBlank() && subtitle != address.name) {
                                Text(
                                    text = subtitle,
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}
