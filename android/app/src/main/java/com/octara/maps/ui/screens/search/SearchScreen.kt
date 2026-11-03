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
    var isLoading by remember { mutableStateOf(false) }

    val context = LocalContext.current
    var currentCoords by remember { mutableStateOf(Pair(48.8566f, 2.3522f)) }

    LaunchedEffect(Unit) {
        val loc = LocationHelper(context).getCurrentLocation()
        if (loc != null) {
            currentCoords = loc
        }
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
                    return@LaunchedEffect
                }

                delay(300)
                isLoading = true
                try {
                    val result = com.octara.maps.utils.Api().searchStopsByName(
                        name = query,
                        limit = 20,
                        cpos = currentCoords
                    )
                    stops = result ?: emptyList()
                } catch (e: Exception) {
                    stops = emptyList()
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