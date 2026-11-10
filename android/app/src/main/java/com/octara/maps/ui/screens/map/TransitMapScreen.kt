package com.octara.maps.ui.screens.map

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material3.BottomSheetScaffold
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SheetValue
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.rememberBottomSheetScaffoldState
import androidx.compose.material3.rememberStandardBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import kotlinx.coroutines.delay
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.dp
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextOverflow
import com.octara.maps.navigation.LocalRouter
import com.octara.maps.ui.components.AppSearchBar
import com.octara.maps.ui.theme.AppDimensions
import org.maplibre.android.geometry.LatLng
import com.octara.maps.managers.FavoritesStopsManager
import com.octara.maps.managers.FavoriteStop
import com.octara.maps.navigation.Screen
import com.octara.maps.types.FavoriteDepartureInfo
import com.octara.maps.types.Route
import com.octara.maps.ui.components.RouteIcon

private fun formatDepartureTime(dep: Number): String {
    val wait = dep.toFloat() / 60.0f
    return when {
        wait <= 1.0f -> "Imminent"
        wait < 60.0f -> "${wait.toInt()} min"
        else -> {
            val hours = (wait / 60).toInt()
            val mins = (wait % 60).toInt()
            if (mins == 0) "${hours} h" else "${hours} h ${mins} min"
        }
    }
}

@Composable
fun TransitMapSheetContent(
    onSearchClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    val router = LocalRouter.current

    val favoritesManager = remember { FavoritesStopsManager() }
    val favoriteStops by favoritesManager.favoriteStops.collectAsState()
    var favoritesStopsNextDepartures by remember {
        mutableStateOf<Map<Pair<String, String>, Map<String, List<FavoriteDepartureInfo>>>>(emptyMap())
    }

    LaunchedEffect(favoriteStops) {
        while (true) {
            if (favoriteStops.isNotEmpty()) {
                try {
                    val pairs = favoriteStops.map { it.stopId to it.routeId }
                    favoritesStopsNextDepartures =
                        com.octara.maps.utils.Api().getNextDepartureDetailsOfMultipleStops(pairs)
                } catch (e: Exception) {
                    e.printStackTrace()
                }
            } else {
                favoritesStopsNextDepartures = emptyMap()
            }
            delay(60_000L)
        }
    }

    Column(
        modifier = modifier
            .fillMaxWidth()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 16.dp, vertical = 8.dp)
            .navigationBarsPadding(),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        AppSearchBar(
            active = false,
            onClick = onSearchClick
        )

        for (favorite in favoriteStops) {
            val nextDeps = favoritesStopsNextDepartures[favorite.stopId to favorite.routeId]
                ?: favoritesStopsNextDepartures[favorite.stopId.removePrefix("IDFM:") to favorite.routeId.removePrefix("IDFM:")]
                ?: emptyMap()
            FavoritesStopsNextDepartures(
                modifier = Modifier.fillMaxWidth(),
                favoriteStop = favorite,
                nextDepartures = nextDeps,
                onClick = {
                    router.navigateTo(
                        Screen.StopInfo(
                            stopId = favorite.stopId,
                            lat = null,
                            lon = null,
                            name = favorite.stopName
                        ),
                        popCurrent = true
                    )
                }
            )
        }
    }
}

@Composable
fun FavoritesStopsNextDepartures(
    modifier: Modifier = Modifier,
    favoriteStop: FavoriteStop,
    nextDepartures: Map<String, List<FavoriteDepartureInfo>>,
    onClick: () -> Unit = {}
) {
    Card(
        onClick = onClick,
        shape = RoundedCornerShape(14.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.35f)
        ),
        modifier = modifier.fillMaxWidth()
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(14.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                RouteIcon(
                    route = Route(
                        id = favoriteStop.routeId,
                        short_name = favoriteStop.routeName,
                        long_name = favoriteStop.routeName,
                        color = favoriteStop.routeColor,
                        picto = favoriteStop.routepicto,
                    ),
                    modifier = Modifier.height(28.dp)
                )
                Column(
                    modifier = Modifier.weight(1f),
                    verticalArrangement = Arrangement.spacedBy(1.dp)
                ) {
                    Text(
                        text = favoriteStop.stopName,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = if (favoriteStop.routeColor.isNotEmpty()) {
                            try {
                                val clean = favoriteStop.routeColor.trim().removePrefix("#")
                                Color((0xFF000000L or clean.toLong(16)).toInt())
                            } catch (_: Exception) {
                                MaterialTheme.colorScheme.onSurface
                            }
                        } else {
                            MaterialTheme.colorScheme.onSurface
                        }
                    )
                }
            }

            HorizontalDivider(
                color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.35f),
                modifier = Modifier.padding(vertical = 2.dp)
            )

            if (nextDepartures.isEmpty()) {
                Text(
                    text = "Aucun départ prévu",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }

            for ((headsign, departures) in nextDepartures) {
                val dep = departures.firstOrNull() ?: continue
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = 3.dp),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Icon(
                        imageVector = Icons.AutoMirrored.Filled.ArrowForward,
                        contentDescription = null,
                        modifier = Modifier.size(14.dp),
                        tint = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Column(
                        modifier = Modifier.weight(1f),
                        verticalArrangement = Arrangement.spacedBy(1.dp)
                    ) {
                        Text(
                            text = if (headsign.isNotEmpty()) headsign else "Direction inconnue",
                            style = MaterialTheme.typography.bodyMedium,
                            fontWeight = FontWeight.Medium,
                            color = MaterialTheme.colorScheme.onSurface,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis
                        )
                        val delayed = dep.is_delayed || dep.delay_sec >= 60
                        if (delayed) {
                            val delayMins =
                                if (dep.delay_minutes > 0) dep.delay_minutes else ((dep.delay_sec + 30) / 60).coerceAtLeast(
                                    1
                                )
                            Row(
                                horizontalArrangement = Arrangement.spacedBy(6.dp),
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Text(
                                    text = "En retard (+$delayMins min)",
                                    style = MaterialTheme.typography.labelSmall,
                                    fontWeight = FontWeight.SemiBold,
                                    color = Color(0xFFD84315)
                                )
                                if (!dep.scheduled_time_string.isNullOrBlank() && !dep.time_string.isNullOrBlank() && dep.scheduled_time_string != dep.time_string) {
                                    Text(
                                        text = dep.time_string,
                                        style = MaterialTheme.typography.labelSmall,
                                        fontWeight = FontWeight.Bold,
                                        color = Color(0xFFD84315)
                                    )
                                }
                            }
                        }
                    }
                    val waitText = formatDepartureTime(dep.wait_sec)
                    val isImminent = (dep.wait_sec.toFloat() / 60.0f) <= 1.0f
                    val isDelayed = dep.is_delayed || dep.delay_sec >= 60
                    val badgeBgColor = when {
                        isDelayed -> Color(0xFFFFF3E0)
                        isImminent -> MaterialTheme.colorScheme.primaryContainer
                        else -> MaterialTheme.colorScheme.surfaceVariant
                    }
                    val badgeTextColor = when {
                        isDelayed -> Color(0xFFD84315)
                        isImminent -> MaterialTheme.colorScheme.onPrimaryContainer
                        else -> MaterialTheme.colorScheme.onSurface
                    }
                    Surface(
                        shape = RoundedCornerShape(8.dp),
                        color = badgeBgColor,
                        modifier = Modifier.padding(start = 4.dp)
                    ) {
                        Text(
                            text = waitText,
                            style = MaterialTheme.typography.labelMedium,
                            fontWeight = if (isImminent || isDelayed) FontWeight.Bold else FontWeight.SemiBold,
                            color = badgeTextColor,
                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                        )
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TransitMapScreen(
    modifier: Modifier = Modifier
) {
    val router = LocalRouter.current
    val configuration = LocalConfiguration.current
    val density = LocalDensity.current

    val currentScreen = router.activeMapScreen
    val isDismissibleState = rememberUpdatedState(currentScreen.isDismissible)

    var targetCoords by remember { mutableStateOf<LatLng?>(null) }
    var targetTitle by remember { mutableStateOf<String?>(null) }

    val favoriteStops by FavoritesStopsManager.favoriteStops.collectAsState()

    val sheetPeekHeight = remember(currentScreen, configuration.screenHeightDp, favoriteStops.isNotEmpty()) {
        currentScreen.sheetPeekHeight(configuration.screenHeightDp.dp)
    }
    val sheetPeekHeightPx = with(density) { sheetPeekHeight.roundToPx() }

    val sheetState = rememberStandardBottomSheetState(
        initialValue = SheetValue.PartiallyExpanded,
        confirmValueChange = { newValue ->
            newValue != SheetValue.Hidden
        },
        skipHiddenState = true
    )
    val scaffoldState = rememberBottomSheetScaffoldState(
        bottomSheetState = sheetState
    )

    val handleClose: () -> Unit = {
        targetCoords = null
        targetTitle = null
        router.goBack()
        Unit
    }

    LaunchedEffect(currentScreen) {
        targetCoords = currentScreen.mapTarget
        targetTitle = currentScreen.mapTitle
        try {
            if (currentScreen.initialSheetValue == SheetValue.Expanded) {
                sheetState.expand()
            } else {
                sheetState.partialExpand()
            }
        } catch (_: Exception) {
        }
    }

    BottomSheetScaffold(
        scaffoldState = scaffoldState,
        sheetShape = AppDimensions.BottomSheetShape,
        sheetPeekHeight = sheetPeekHeight,
        sheetShadowElevation = AppDimensions.BottomSheetElevation,
        sheetContent = {
            key(currentScreen) {
                currentScreen.SheetContent(
                    sheetState = sheetState,
                    onClose = handleClose,
                    onUpdateTarget = { coords, title ->
                        if (targetCoords == null) {
                            targetCoords = coords
                            targetTitle = title
                        }
                    }
                )
            }
        },
        modifier = modifier.fillMaxSize()
    ) {
        Box(modifier = Modifier.fillMaxSize()) {
            TransitMap(
                modifier = Modifier.fillMaxSize(),
                currentScreen = currentScreen,
                selectedStopDetails = router.selectedStopDetails,
                cameraTarget = targetCoords,
                cameraZoom = currentScreen.cameraZoom,
                bottomPaddingPx = sheetPeekHeightPx,
                markerPosition = targetCoords,
                markerTitle = targetTitle,
                activeJourney = router.activeJourney
            )

            AnimatedVisibility(
                visible = currentScreen.showBackButton,
                enter = fadeIn(),
                exit = fadeOut(),
                modifier = Modifier
                    .statusBarsPadding()
                    .padding(start = 16.dp, top = 8.dp)
                    .align(Alignment.TopStart)
            ) {
                Surface(
                    shape = CircleShape,
                    color = MaterialTheme.colorScheme.surface.copy(alpha = 0.92f),
                    shadowElevation = 4.dp
                ) {
                    IconButton(
                        onClick = handleClose,
                        modifier = Modifier.size(42.dp)
                    ) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = "Retour",
                            tint = MaterialTheme.colorScheme.onSurface
                        )
                    }
                }
            }
        }
    }
}
