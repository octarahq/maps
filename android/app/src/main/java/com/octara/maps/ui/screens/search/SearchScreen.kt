package com.octara.maps.ui.screens.search

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
import androidx.compose.material.icons.filled.AirlineSeatReclineExtra
import androidx.compose.material.icons.filled.DirectionsBus
import androidx.compose.material.icons.filled.LocationOn
import androidx.compose.material.icons.filled.Subway
import androidx.compose.material.icons.filled.Train
import androidx.compose.material.icons.filled.Tram
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier as ComposeModifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.octara.maps.navigation.LocalRouter
import com.octara.maps.navigation.Router
import com.octara.maps.navigation.Screen
import com.octara.maps.ui.components.AppSearchBar
import com.octara.maps.utils.LocationHelper
import com.octara.maps.utils.LocationManager
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.delay
import com.octara.maps.ui.components.getStopIconResource

@Composable
fun SearchScreen(
    onBack: () -> Unit,
    modifier: ComposeModifier = ComposeModifier
) {
    val router = LocalRouter.current
    var query by remember { mutableStateOf("") }
    var stops by remember { mutableStateOf<List<com.octara.maps.types.ParentStop>>(emptyList()) }
    var addresses by remember { mutableStateOf<List<com.octara.maps.utils.Api.AddressResult>>(emptyList()) }
    var isLoading by remember { mutableStateOf(false) }

    val context = LocalContext.current
    val currentCoords by LocationManager.currentLocation.collectAsState()

    LaunchedEffect(Unit) {
        LocationManager.fetchCurrentLocation(context)
    }

    Surface(
        modifier = modifier.fillMaxSize(),
        color = MaterialTheme.colorScheme.surface
    ) {
        Column(
            modifier = ComposeModifier
                .fillMaxSize()
                .statusBarsPadding()
        ) {
            AppSearchBar(
                query = query,
                onQueryChange = { query = it },
                onSearch = {
                    if (query.startsWith("IDFM:")) {
                        val id = query.removePrefix("IDFM:")
                        router.navigateTo(Screen.StopInfo(id), popCurrent = true)
                    }
                },
                active = true,
                autoFocus = true,
                onBack = onBack,
                modifier = ComposeModifier.padding(horizontal = 16.dp, vertical = 8.dp),
                isLoading = isLoading
            )

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
                    val api = com.octara.maps.utils.Api()
                    val (stopsRes, addressesRes) = coroutineScope {
                        val s = async {
                            runCatching { api.searchStopsByName(name = query, limit = 15, cpos = currentCoords) }.getOrDefault(emptyList())
                        }
                        val a = async {
                            runCatching { api.searchAddresses(query = query, limit = 10, cpos = currentCoords) }.getOrDefault(emptyList())
                        }
                        s.await() to a.await()
                    }
                    stops = stopsRes
                    addresses = addressesRes
                } catch (e: Exception) {
                    stops = emptyList()
                    addresses = emptyList()
                } finally {
                    isLoading = false
                }
            }

            LazyColumn(
                modifier = ComposeModifier
                    .fillMaxWidth()
                    .weight(1f)
                    .padding(horizontal = 16.dp)
            ) {
                items(stops) { stop ->
                    SearchResultItem(stop = stop, router = router)
                }
                items(addresses) { address ->
                    AddressSearchResultItem(address = address, router = router)
                }
            }
        }
    }
}

@Composable
fun SearchResultItem(
    stop: com.octara.maps.types.ParentStop,
    router: Router = LocalRouter.current
) {
    Row(
        modifier = ComposeModifier
            .fillMaxWidth()
            .clickable {
                router.navigateTo(
                    Screen.StopInfo(
                        stopId = stop.id,
                        lat = stop.lat.toDouble(),
                        lon = stop.lon.toDouble(),
                        name = stop.name
                    ),
                    popCurrent = true
                )
            }
            .padding(vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Box(
            modifier = ComposeModifier
                .size(40.dp)
                .clip(RoundedCornerShape(10.dp))
                .background(MaterialTheme.colorScheme.primary.copy(alpha = 0.1f)),
            contentAlignment = Alignment.Center
        ) {
            Icon(
                imageVector = getStopIconResource(stop.type),
                contentDescription = "Icône de transport",
                tint = MaterialTheme.colorScheme.primary,
            )
        }
        Text(
            text = stop.name,
            style = MaterialTheme.typography.bodyLarge,
            modifier = ComposeModifier.padding(vertical = 8.dp)
        )
    }
}

@Composable
fun AddressSearchResultItem(
    address: com.octara.maps.utils.Api.AddressResult,
    router: Router = LocalRouter.current
) {
    Row(
        modifier = ComposeModifier
            .fillMaxWidth()
            .clickable {
                router.navigateTo(
                    Screen.JourneyPlanner(
                        toStopName = address.name,
                        toLat = address.lat.toDouble(),
                        toLon = address.lon.toDouble()
                    ),
                    popCurrent = true
                )
            }
            .padding(vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Box(
            modifier = ComposeModifier
                .size(40.dp)
                .clip(RoundedCornerShape(10.dp))
                .background(MaterialTheme.colorScheme.secondary.copy(alpha = 0.1f)),
            contentAlignment = Alignment.Center
        ) {
            Icon(
                imageVector = androidx.compose.material.icons.Icons.Default.LocationOn,
                contentDescription = "Adresse",
                tint = MaterialTheme.colorScheme.secondary,
            )
        }
        Column {
            Text(
                text = address.name,
                style = MaterialTheme.typography.bodyLarge
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
